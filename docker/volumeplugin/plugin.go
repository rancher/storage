package volumeplugin

import (
	"os"
	"path/filepath"

	"k8s.io/kubernetes/pkg/util/exec"
	"k8s.io/kubernetes/pkg/util/mount"

	"github.com/Sirupsen/logrus"
	dockerClient "github.com/docker/engine-api/client"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/v2"
)

var errNoSuchVolume = errors.New("No such volume")

const (
	k8sFsType      = "kubernetes.io/fsType"
	fsType         = "fs-type"
	RancherUUID    = "rancher-uuid"
	DefaultBasedir = "/var/lib/rancher/volumes"
	DefaultFsType  = "ext4"
	DefaultScope   = "global"
)

func NewRancherStorageDriver(driver string, client *client.RancherClient, cli *dockerClient.Client) (*RancherStorageDriver, error) {
	state, err := NewRancherState(driver, client)
	if err != nil {
		return nil, err
	}
	d := &RancherStorageDriver{
		DriverName:      driver,
		Basedir:         DefaultBasedir,
		Scope:           DefaultScope,
		CreateSupported: true,
		Command:         driver,
		client:          client,
		state:           state,
		mounter:         &mount.SafeFormatAndMount{Interface: mount.New(), Runner: exec.New()},
		FsType:          DefaultFsType,
		cli:             cli,
	}
	if err := d.init(); err != nil {
		return nil, errors.Wrap(err, "Failed to initialize")
	}
	return d, nil
}

type RancherStorageDriver struct {
	DriverName      string
	Basedir         string
	Scope           string
	CreateSupported bool
	Command         string
	client          *client.RancherClient
	state           *RancherState
	mounter         *mount.SafeFormatAndMount
	FsType          string
	cli             *dockerClient.Client
}

func (d *RancherStorageDriver) init() error {
	_, err := d.exec("init")
	return err
}

func (d *RancherStorageDriver) Create(request volume.Request) volume.Response {
	logRequest("create", &request)

	response := volume.Response{}
	defer logResponse("create", &response)

	if created, err := d.state.IsCreated(request.Name); err != nil {
		response.Err = err.Error()
		return response
	} else if created {
		return response
	}

	result := request.Options
	if d.CreateSupported {
		cmdOutput, err := d.exec("create", toArgs(request.Name, request.Options))
		if err != nil {
			response.Err = err.Error()
			return response
		}
		result = fold(result, cmdOutput.Options)
	}

	if err := d.state.Save(request.Name, result); err != nil {
		logrus.Errorf("Save volume name=%s failed, err: %s", request.Name, err)
		d.exec("delete", toArgs(request.Name, result))
		response.Err = err.Error()
		return response
	}

	return response
}

func (d *RancherStorageDriver) List(request volume.Request) volume.Response {
	//logRequest("list", &request)

	response := volume.Response{}
	//defer logResponse("list", &response)

	volumes, err := d.state.List()
	if err != nil {
		response.Err = err.Error()
	} else {
		response.Volumes = volumes
	}

	return response
}

func (d *RancherStorageDriver) Get(request volume.Request) volume.Response {
	//logRequest("get", &request)

	response := volume.Response{}
	//defer logResponse("get", &response)

	vol, _, err := d.state.Get(request.Name)
	if err != nil {
		response.Err = err.Error()
		return response
	}
	if vol != nil {
		response.Volume = vol
	}

	return response
}

func (d *RancherStorageDriver) Remove(request volume.Request) volume.Response {
	logRequest("remove", &request)

	response := volume.Response{}
	defer logResponse("remove", &response)

	_, rVol, err := d.state.Get(request.Name)
	if err != nil {
		response.Err = err.Error()
		return response
	}

	_, err = d.exec("delete", toArgs(request.Name, getOptions(rVol)))
	if err != nil {
		response.Err = err.Error()
		return response
	}

	err = d.state.Delete(request.Name)
	if err != nil {
		response.Err = err.Error()
	}

	return response
}

func (d *RancherStorageDriver) isMounted(path string) (bool, error) {
	mounts, err := d.mounter.List()
	if err != nil {
		return false, err
	}
	for _, mount := range mounts {
		if mount.Path == path {
			return true, nil
		}
	}

	return false, nil
}

func (d *RancherStorageDriver) Mount(request volume.MountRequest) volume.Response {
	logrus.WithFields(logrus.Fields{
		"name": request.Name,
	}).Info("mount.request")

	response := volume.Response{}
	defer logResponse("mount", &response)

	_, rVol, err := d.state.Get(request.Name)
	if err != nil {
		response.Err = err.Error()
		return response
	}

	mntDest := d.getMntDest(request.Name)
	if mounted, err := d.isMounted(mntDest); err != nil {
		response.Err = errors.Wrap(err, "checking mounts").Error()
		return response
	} else if mounted {
		logrus.Infof("%s already mounted on %s", request.Name, mntDest)
		response.Mountpoint = mntDest
		return response
	}

	opts := toArgs(request.Name, getOptions(rVol))
	cmdOutput, err := d.exec("attach", opts)
	if err != nil && err != errNotSupported {
		logrus.Errorf("Failed to attach %s: %v", request.Name, err)
		response.Err = err.Error()
		return response
	}

	os.MkdirAll(mntDest, 0750)
	if _, err := d.exec("mount", mntDest, cmdOutput.Device, opts); err != nil {
		logrus.Errorf("Failed to mount %s: %v", request.Name, err)
		response.Err = err.Error()
		return response
	}

	response.Mountpoint = mntDest
	return response
}

func (d *RancherStorageDriver) getFsType(vol *client.Volume) string {
	fsType, _ := vol.DriverOpts[fsType].(string)
	if fsType == "" {
		fsType, _ = vol.DriverOpts[k8sFsType].(string)
	}
	if fsType == "" {
		fsType = d.FsType
	}
	return fsType
}

func (d *RancherStorageDriver) Unmount(request volume.UnmountRequest) volume.Response {
	logrus.WithFields(logrus.Fields{
		"name": request.Name,
	}).Info("unmount.request")

	response := volume.Response{}
	defer logResponse("unmount", &response)

	if err := d.unmount(d.getMntDest(request.Name)); err != nil {
		response.Err = errors.Wrap(err, "unmount").Error()
	}

	return response
}

func (d *RancherStorageDriver) unmount(mntDest string) error {
	device, refCount, err := mount.GetDeviceNameFromMount(d.mounter, mntDest)
	if err != nil {
		return errors.Wrapf(err, "find device %s", mntDest)
	}

	if _, err := d.exec("unmount", mntDest); err != nil {
		return errors.Wrapf(err, "umount %s", mntDest)
	}

	if refCount != 1 {
		return nil
	}

	if _, err := d.exec("detach", device); err != nil && err != errNotSupported {
		return errors.Wrapf(err, "detach %s", device)
	}

	if notmnt, err := d.mounter.IsLikelyNotMountPoint(mntDest); err != nil {
		return errors.Wrap(err, "Lookup mount")
	} else if notmnt {
		if err := os.Remove(mntDest); err != nil {
			return errors.Wrapf(err, "delete %s", mntDest)
		}
	}

	return nil
}

func (d *RancherStorageDriver) Path(request volume.Request) volume.Response {
	return volume.Response{
		Mountpoint: d.getMntDest(request.Name),
	}
}

func (d *RancherStorageDriver) Capabilities(volume.Request) volume.Response {
	return volume.Response{
		Capabilities: volume.Capability{
			Scope: d.Scope,
		},
	}
}

func (d *RancherStorageDriver) getMntDest(name string) string {
	return filepath.Join(d.getMntRoot(), name)
}

func (d *RancherStorageDriver) getMntRoot() string {
	return filepath.Join(d.Basedir, d.DriverName)
}
