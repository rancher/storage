package volumeplugin

import (
	"encoding/json"
	"fmt"
	"time"

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
	"inactive":     true,
	"active":       true,
	"activating":   true,
	"deactivating": true,
	"detached":     true,
	"removing":     true,
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
		return nil, errors.Wrap(err, "getting driver ID")
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
	_, _, err := r.Get(name)
	if err == errNoSuchVolume {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *RancherState) Save(name string, options map[string]string, try int) error {
	// Wait for the volume to be created by Rancher
	_, vol, err := r.getAny(name)
	for tries := 1; err != nil; tries++ {
		if tries == 30 {
			return errors.Wrap(err, "Max tries reached")
		}

		_, vol, err = r.getAny(name)

		if err != nil {
			if err == errNoSuchVolume {
				time.Sleep(time.Second)
				continue
			} else {
				return err
			}
		} else {
			break
		}
	}

	_, err = r.client.Volume.Update(vol, &client.Volume{
		Name:            name,
		Driver:          r.driver,
		StorageDriverId: r.driverID,
		DriverOpts:      toMapInterface(options),
		HostId:          r.hostID,
	})

	if apiErr, ok := err.(*client.ApiError); ok && apiErr.StatusCode == 409 {
		if try < 5 {
			try++
			wait := try * 2
			logrus.Warnf("409 Conflict while updating volume %s. Sleeping %s and retrying.", vol.Id, wait)
			time.Sleep(time.Duration(wait) * time.Second)
			return r.Save(name, options, try)
		}
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
		if isCreated(vol) {
			result = append(result, volToVol(vol))
		}
	}

	return result, nil
}

func isCreated(vol client.Volume) bool {
	return goodStates[vol.State]
}

func (r *RancherState) getAny(name string) (*volume.Volume, *client.Volume, error) {
	vols, err := r.client.Volume.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"name":            name,
			"removed_null":    "true",
			"storageDriverId": r.driverID,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	if len(vols.Data) == 0 {
		return nil, nil, errNoSuchVolume
	}

	return volToVol(vols.Data[0]), &vols.Data[0], nil
}

func (r *RancherState) Get(name string) (*volume.Volume, *client.Volume, error) {
	vols, err := r.client.Volume.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"name":            name,
			"removed_null":    "true",
			"storageDriverId": r.driverID,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	if len(vols.Data) > 1 {
		logrus.Warnf("%d volumes with name=%s found in Rancher", len(vols.Data), name)
	}

	for _, vol := range vols.Data {
		if isCreated(vol) {
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
