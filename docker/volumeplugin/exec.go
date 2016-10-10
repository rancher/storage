package volumeplugin

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
)

const (
	statusSuccess      = "Success"
	statusFailure      = "Failure"
	statusNotSupported = "Not supported"
)

var errNotSupported = errors.New("Unsupported Operation")

type CmdOutput struct {
	Status  string
	Message string
	Options map[string]string
	Device  string `json:"device"`
}

func (d *RancherStorageDriver) exec(command string, args ...string) (CmdOutput, error) {
	result := CmdOutput{}
	buf := &bytes.Buffer{}
	cmd := exec.Command(d.Command, append([]string{command}, args...)...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = buf

	if err := cmd.Run(); err != nil {
		if json.Unmarshal(buf.Bytes(), &result) == nil && result.Message != "" {
			return result, errors.New(result.Message)
		}
		return result, err
	}

	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		return result, err
	}

	if result.Status == statusFailure {
		return result, errors.New(result.Message)
	}

	if result.Status == statusNotSupported {
		return result, errNotSupported
	}

	return result, nil
}
