package volumeplugin

import (
	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/rancher/go-rancher/v2"
)

func volErr(err error) volume.Response {
	return volume.Response{
		Err: err.Error(),
	}
}

func errorToResponse(err error) volume.Response {
	logrus.Errorf("Error response: %v", err)
	return volume.Response{
		Err: err.Error(),
	}
}

func getOptions(vol *client.Volume) map[string]string {
	if vol == nil {
		return nil
	}
	result := map[string]string{}
	for k, v := range vol.DriverOpts {
		result[k] = fmt.Sprint(v)
	}
	return result
}

func volToResponse(err error, vol *volume.Volume) volume.Response {
	if err != nil {
		return volErr(err)
	}

	logrus.Infof("Response: %v", vol)
	return volume.Response{
		Volume: vol,
	}
}

func fold(data ...map[string]string) map[string]string {
	result := map[string]string{}
	for _, d := range data {
		for k, v := range d {
			result[k] = v
		}
	}
	return result
}

func toArgs(name string, data map[string]string) string {
	if data == nil {
		data = map[string]string{}
	}
	data["name"] = name
	data["rancher"] = "true"
	bytes, err := json.Marshal(data)
	if err != nil {
		// This really shouldn't ever happen
		panic(err)
	}
	return string(bytes)
}
