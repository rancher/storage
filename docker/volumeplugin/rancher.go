package volumeplugin

import (
	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher-metadata/metadata"
	"github.com/rancher/go-rancher/v2"
)

const (
	metadataURL = "http://169.254.169.250/2015-12-19"
)

var goodStates = map[string]bool{
	"active":       true,
	"activating":   true,
	"deactivating": true,
	"detached":     true,
}

type RancherState struct {
	client   *client.RancherClient
	driver   string
	hostID   string
	driverID string
}

func NewRancherState(driver string, client *client.RancherClient) (*RancherState, error) {
	host, err := getHostID(client)
	if err != nil {
		return nil, errors.Wrap(err, "getting host ID")
	}
	driverID, err := getDriverID(driver, client)
	if err != nil {
		return nil, errors.Wrap(err, "getting host ID")
	}

	logrus.Infof("Running on host %s(%s) with driver %s(%s)", host.Hostname, host.Id, driver, driverID)
	return &RancherState{
		client:   client,
		driver:   driver,
		hostID:   host.Id,
		driverID: driverID,
	}, nil
}

func (r *RancherState) IsCreated(name string) (bool, error) {
	_, vol, err := r.Get(name)
	if err == errNoSuchVolume {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return isCreated(r.driver, *vol), nil
}

func (r *RancherState) Save(name string, options map[string]string) error {
	_, vol, err := r.Get(name)
	if err == errNoSuchVolume {
		_, err = r.client.Volume.Create(&client.Volume{
			Name:            name,
			StorageDriverId: r.driverID,
			DriverOpts:      toMapInterface(options),
			HostId:          r.hostID,
		})
		return err
	} else if err == nil {
		_, err = r.client.Volume.Update(vol, &client.Volume{
			Name:            name,
			Driver:          r.driver,
			StorageDriverId: r.driverID,
			DriverOpts:      toMapInterface(options),
			HostId:          r.hostID,
		})
		return err
	}
	return err
}

func (r *RancherState) List() ([]*volume.Volume, error) {
	vols, err := r.client.Volume.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"removed_null":    "true",
			"limit":           "-1",
			"storageDriverId": r.driverID,
		},
	})
	if err != nil {
		return nil, err
	}
	result := []*volume.Volume{}
	for _, vol := range vols.Data {
		if isCreated(r.driver, vol) {
			result = append(result, volToVol(vol))
		}
	}

	return result, nil
}

func isCreated(driver string, vol client.Volume) bool {
	return goodStates[vol.State]
}

func (r *RancherState) Get(name string) (*volume.Volume, *client.Volume, error) {
	vols, err := r.client.Volume.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"name":         name,
			"removed_null": "true",
		},
	})
	if err != nil {
		return nil, nil, err
	}

	for _, vol := range vols.Data {
		if isCreated(r.driver, vol) {
			return volToVol(vol), &vol, nil
		}
	}

	return nil, nil, errNoSuchVolume
}

func volToVol(vol client.Volume) *volume.Volume {
	result := &volume.Volume{
		Name:   vol.Name,
		Status: map[string]interface{}{},
	}
	bytes, err := json.Marshal(vol)
	if err == nil {
		json.Unmarshal(bytes, &result.Status)
	}
	return result
}

func toMapInterface(data map[string]string) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range data {
		result[k] = v
	}
	return result
}

func getDriverID(driver string, c *client.RancherClient) (string, error) {
	drivers, err := c.StorageDriver.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"name":         driver,
			"removed_null": true,
		},
	})
	if err != nil {
		return "", err
	}
	if len(drivers.Data) != 1 {
		return "", fmt.Errorf("%s is not a driver registered with the current Rancher environment", driver)
	}
	return drivers.Data[0].Id, nil
}

func getHostID(c *client.RancherClient) (*client.Host, error) {
	m, err := metadata.NewClientAndWait(metadataURL)
	if err != nil {
		return nil, errors.Wrap(err, "initializing metadata")
	}
	mHost, err := m.GetSelfHost()
	if err != nil {
		return nil, err
	}
	hosts, err := c.Host.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"uuid": mHost.UUID,
		},
	})
	if len(hosts.Data) != 1 {
		return nil, fmt.Errorf("Failed to find current host %s, got %d host(s)", mHost.UUID, len(hosts.Data))
	}
	return &hosts.Data[0], nil
}
