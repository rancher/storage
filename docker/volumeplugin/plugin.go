package volumeplugin

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"k8s.io/kubernetes/pkg/util/exec"
	"k8s.io/kubernetes/pkg/util/mount"

	"github.com/Sirupsen/logrus"
	dockerClient "github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/events"
	"github.com/docker/engine-api/types/filters"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/v2"
	"golang.org/x/net/context"
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
	d.kickGC()
	go d.watchContainerDeletes()
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
	mountLock       sync.Mutex
}

func (d *RancherStorageDriver) init() error {
	_, err := d.exec("init")
	return err
}

func (d *RancherStorageDriver) Create(request volume.Request) volume.Response {
	logrus.Infof("Docker Create request: %v", request)

	if created, err := d.state.IsCreated(request.Name); err != nil {
		return volErr(err)
	} else if created {
		return volume.Response{}
	}

	result := request.Options
	if d.CreateSupported {
		cmdOutput, err := d.exec("create", toArgs(request.Name, request.Options))
		if err != nil {
			return volErr(err)
		}
		result = fold(result, cmdOutput.Options)
	}

	if err := d.state.Save(request.Name, result); err != nil {
		logrus.Debugf("Save volume name=%s failed, err: %s", request.Name, err)
		d.exec("delete", toArgs(request.Name, result))
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
	logrus.Infof("Docker Remove request: %v", request)

	_, rVol, err := d.state.Get(request.Name)
	if err == errNoSuchVolume {
		return volume.Response{}
	} else if err != nil {
		return volErr(err)
	}

	// Docker removal is fake, unless Rancher initiated removal of resource, then we do it.
	if rVol.State == "removing" {
		_, err := d.exec("delete", toArgs(request.Name, getOptions(rVol)))
		if err != nil {
			return volErr(err)
		}
	}

	return volume.Response{}
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
	d.mountLock.Lock()
	defer d.mountLock.Unlock()

	logrus.Infof("Docker Mount request: %v", request)
	_, rVol, err := d.state.Get(request.Name)
	if err != nil {
		return volErr(err)
	}

	mntDest := d.getMntDest(request.Name)
	if mounted, err := d.isMounted(mntDest); err != nil {
		return volErr(errors.Wrap(err, "checking mounts"))
	} else if mounted {
		logrus.Infof("%s already mounted on %s", request.Name, mntDest)
		return volume.Response{
			Mountpoint: mntDest,
		}
	}

	opts := toArgs(request.Name, getOptions(rVol))
	cmdOutput, err := d.exec("attach", opts)
	if err != nil && err != errNotSupported {
		logrus.Errorf("Failed to attach %s: %v", request.Name, err)
		return volErr(err)
	}

	os.MkdirAll(mntDest, 0750)
	if _, err := d.exec("mount", mntDest, cmdOutput.Device, opts); err == errNotSupported {
		fsType := d.getFsType(rVol)
		logrus.Infof("Formatting and mounting %s", cmdOutput.Device)
		if err := d.mounter.FormatAndMount(cmdOutput.Device, mntDest, fsType, []string{}); err != nil {
			logrus.Errorf("Failed to format and mount %s: %v", request.Name, err)
			return volErr(errors.Wrap(err, "mount"))
		}
	} else if err != nil {
		logrus.Errorf("Failed to mount %s: %v", request.Name, err)
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
	d.kickGC()
	return volume.Response{}
}

func (d *RancherStorageDriver) unmount(mntDest string) error {
	d.mountLock.Lock()
	defer d.mountLock.Unlock()

	logrus.Infof("Unmounting %s", mntDest)
	device, refCount, err := mount.GetDeviceNameFromMount(d.mounter, mntDest)
	if err != nil {
		return errors.Wrapf(err, "find device %s", mntDest)
	}

	if _, err := d.exec("unmount", mntDest); err == errNotSupported {
		if err := d.mounter.Unmount(mntDest); err != nil {
			return errors.Wrapf(err, "umount with mounter %s", mntDest)
		}
	} else if err != nil {
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

func (d *RancherStorageDriver) kickGC() {
	go func() {
		time.Sleep(time.Second)
		if err := d.gc(); err != nil {
			logrus.Errorf("Failed to run GC: %v", err)
		}
	}()
}

func (d *RancherStorageDriver) gc() error {
	mntRoot := d.getMntRoot()
	mounts, err := d.mounter.List()
	if err != nil {
		return err
	}

	toUnmount := map[string]bool{}
	toCheck := map[string]bool{}
	for _, mount := range mounts {
		if strings.HasPrefix(mount.Path, mntRoot) {
			toCheck[mount.Path] = true
		}
	}

	if len(toCheck) == 0 {
		return nil
	}

	knownToDocker := map[string]bool{}
	volumeResp, err := d.cli.VolumeList(context.Background(), filters.NewArgs())
	if err != nil {
		return err
	}
	for _, volume := range volumeResp.Volumes {
		if volume.Driver == d.DriverName {
			knownToDocker[d.getMntDest(volume.Name)] = true
		}
	}

	args := filters.NewArgs()
	args.Add("dangling", "true")
	volumeResp, err = d.cli.VolumeList(context.Background(), args)
	if err != nil {
		return err
	}
	for _, volume := range volumeResp.Volumes {
		if volume.Driver == d.DriverName {
			dest := d.getMntDest(volume.Name)
			if toCheck[dest] {
				toUnmount[d.getMntDest(volume.Name)] = true
			}
		}
	}

	for mount := range toCheck {
		if !knownToDocker[mount] {
			logrus.Errorf("Mounted but not registered in Docker: %s", mount)
			toUnmount[mount] = true
		}
	}

	var lastErr error
	for mnt := range toUnmount {
		if err := d.unmount(mnt); err != nil {
			lastErr = err
			logrus.Errorf("Failed to unmount %s: %v", mnt, err)
		}
	}

	return lastErr
}

func (d *RancherStorageDriver) watchContainerDeletes() error {
	for {
		reader, err := d.cli.Events(context.Background(), types.EventsOptions{})
		if err != nil {
			return err
		}
		defer reader.Close()

		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			var event events.Message
			if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
				logrus.Errorf("Failed to unmarshal %s: %v", scanner.Text(), err)
			}
			if event.Status == "destroy" {
				logrus.Infof("container %s destroyed", event.ID)
				d.kickGC()
			}
		}
		time.Sleep(2 * time.Second)
	}
}
