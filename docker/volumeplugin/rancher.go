package volumeplugin

import (
	"encoding/json"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/rancher/go-rancher/v2"
)

type RancherState struct {
	client *client.RancherClient
	driver string
}

func (r *RancherState) Save(name string, options map[string]string) error {
	_, vol, err := r.Get(name)
	if err == errNoSuchVolume {
		_, err = r.client.Volume.Create(&client.Volume{
			Name:       name,
			Driver:     r.driver,
			DriverOpts: toMapInterface(options),
		})
		return err
	} else if err == nil {
		_, err = r.client.Volume.Update(vol, &client.Volume{
			Name:       name,
			Driver:     r.driver,
			DriverOpts: toMapInterface(options),
		})
		return err
	}
	return err
}

func (r *RancherState) List() ([]*volume.Volume, error) {
	// TODO: Optimize this
	vols, err := r.client.Volume.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"removed_null": "true",
			"limit":        "-1",
		},
	})
	if err != nil {
		return nil, err
	}
	result := []*volume.Volume{}
	for _, vol := range vols.Data {
		if vol.Driver == r.driver {
			result = append(result, volToVol(vol))
		}
	}

	return result, nil
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
		if vol.Driver == r.driver {
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
