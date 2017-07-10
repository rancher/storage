package volumeplugin

import (
	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/v2"
)

func logRequest(action string, request *volume.Request) {
	fields := logrus.Fields{}
	if request.Name != "" {
		fields["name"] = request.Name
	}
	if len(request.Options) > 0 {
		fields["options"] = request.Options
	}
	logrus.WithFields(fields).Infof("%s.request", action)
}

func logResponse(action, name string, response *volume.Response, output *CmdOutput) {
	fields := logrus.Fields{}
	fields["name"] = name
	if response.Mountpoint != "" {
		fields["mountpoint"] = response.Mountpoint
	}
	if output.Message != "" {
		fields["message"] = output.Message
	}
	if output.Status != "" {
		fields["status"] = output.Status
	}
	if response.Err != "" {
		fields["error"] = response.Err
		logrus.WithFields(fields).Errorf("%s.response", action)
	} else {
		logrus.WithFields(fields).Infof("%s.response", action)
	}
}

func volErr2(message string, err error) volume.Response {
	return volume.Response{
		Err: errors.Wrap(err, message).Error(),
	}
}

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
