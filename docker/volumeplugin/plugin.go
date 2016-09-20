package volumeplugin

import (
	"os"
	"path/filepath"

	"k8s.io/kubernetes/pkg/util/exec"
	"k8s.io/kubernetes/pkg/util/mount"

	"github.com/Sirupsen/logrus"
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

func NewRancherStorageDriver(driver string, client *client.RancherClient) *RancherStorageDriver {
	return &RancherStorageDriver{
		DriverName:      driver,
		Basedir:         DefaultBasedir,
		Scope:           DefaultScope,
		CreateSupported: true,
		Command:         driver,
		client:          client,
		state: &RancherState{
			client: client,
			driver: driver,
		},
		mounter: &mount.SafeFormatAndMount{Interface: mount.New(), Runner: exec.New()},
		FsType:  DefaultFsType,
	}
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
}

func (d *RancherStorageDriver) Create(request volume.Request) volume.Response {
	logrus.Infof("Docker create request: %v", request)

	result := request.Options
	if d.CreateSupported {
		cmdOutput, err := d.exec("create", toArgs(request.Name, request.Options))
		if err != nil {
			return volErr(err)
		}
		result = fold(result, cmdOutput.Options)
	}

	if err := d.state.Save(request.Name, result); err != nil {
		return volErr(err)
	}

	return volume.Response{}
}

func (d *RancherStorageDriver) List(request volume.Request) volume.Response {
	logrus.Infof("Docker List request: %v", request)

	volumes, err := d.state.List()
	if err != nil {
		return volErr(err)
	}
	return volume.Response{
		Volumes: volumes,
	}
}

func (d *RancherStorageDriver) Get(request volume.Request) volume.Response {
	logrus.Infof("Docker Get request: %v", request)

	vol, _, err := d.state.Get(request.Name)
	return volToResponse(err, vol)
}

func (d *RancherStorageDriver) Remove(request volume.Request) volume.Response {
	_, rVol, err := d.state.Get(request.Name)
	if err == errNoSuchVolume {
		return volume.Response{}
	} else if err != nil {
		return volErr(err)
	}

	if rVol.State == "removing" {
		_, err := d.exec("delete", toArgs(request.Name, getOptions(rVol)))
		if err != nil {
			return volErr(err)
		}
	}

	return volume.Response{}
}

func (d *RancherStorageDriver) Mount(request volume.MountRequest) volume.Response {
	logrus.Infof("Docker Mount request: %v", request)
	_, rVol, err := d.state.Get(request.Name)
	if err != nil {
		return volErr(err)
	}

	opts := toArgs(request.Name, getOptions(rVol))
	cmdOutput, err := d.exec("attach", opts)
	if err != nil && err != errNotSupported {
		return volErr(err)
	}

	mntDest := d.getMntDest(request.Name)
	os.MkdirAll(mntDest, 0750)
	if _, err := d.exec("mount", mntDest, cmdOutput.Device, opts); err == errNotSupported {
		fsType := d.getFsType(rVol)
		if err := d.mounter.FormatAndMount(cmdOutput.Device, mntDest, fsType, []string{}); err != nil {
			return volErr(errors.Wrap(err, "mount"))
		}
	} else if err != nil {
		return volErr(err)
	}

	return volume.Response{
		Mountpoint: mntDest,
	}
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
	logrus.Infof("Docker Unmount request: %v", request)
	mntDest := d.getMntDest(request.Name)
	device, refCount, err := mount.GetDeviceNameFromMount(d.mounter, mntDest)
	if err != nil {
		return volErr(errors.Wrapf(err, "find device %s", request.Name))
	}

	if _, err := d.exec("unmount", mntDest); err == errNotSupported {
		if err := d.mounter.Unmount(mntDest); err != nil {
			return volErr(errors.Wrapf(err, "umount with mounter %s", mntDest))
		}
	} else if err != errNotSupported {
		return volErr(errors.Wrapf(err, "umount %s", mntDest))
	}

	if refCount != 1 {
		return volume.Response{}
	}

	if _, err := d.exec("detach", device); err != nil && err != errNotSupported {
		return volErr(errors.Wrapf(err, "detach %s", device))
	}

	if notmnt, err := d.mounter.IsLikelyNotMountPoint(mntDest); err != nil {
		return volErr(errors.Wrap(err, "Lookup mount"))
	} else if notmnt {
		if err := os.Remove(mntDest); err != nil {
			volErr(errors.Wrapf(err, "delete %s", mntDest))
		}
	}

	return volume.Response{}
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
	return filepath.Join(d.Basedir, d.DriverName, name)
}
