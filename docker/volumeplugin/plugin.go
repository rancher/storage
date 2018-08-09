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
	"github.com/docker/docker/pkg/locker"
	dockerClient "github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/events"
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
	DefaultScope   = "flex"
	state          = "state"
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
		SaveOnAttach:    false,
		mountMap:        map[string]map[string]struct{}{},
		lock:            locker.New(),
		Rancher:         rancherDrivers[driver],
	}
	if err := d.init(); err != nil {
		return nil, errors.Wrap(err, "Failed to initialize")
	}
	go syncMountMap(d, cli)
	d.kickGC()
	go d.watchContainerEvents()
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
	SaveOnAttach    bool
	mountMap        map[string]map[string]struct{}
	mountMapLock    sync.RWMutex
	lock            *locker.Locker
	Rancher         bool
}

func (d *RancherStorageDriver) init() error {
	_, err := d.exec("init")
	return err
}

func (d *RancherStorageDriver) Validate() error {
	_, err := d.exec("validate")
	return err
}

func (d *RancherStorageDriver) Create(request volume.Request) volume.Response {
	// we need to lock the name to make create idempotency
	if d.Rancher {
		d.lock.Lock(request.Name)
		defer d.lock.Unlock(request.Name)
	}

	logRequest("create", &request)

	response := volume.Response{}
	output := &CmdOutput{}
	defer logResponse("create", request.Name, &response, output)

	if created, err := d.state.IsCreated(request.Name); err != nil {
		response.Err = err.Error()
		return response
	} else if created {
		return response
	}

	result := request.Options
	if d.CreateSupported {
		var err error
		*output, err = d.exec("create", toArgs(request.Name, request.Options))
		if err != nil {
			response.Err = err.Error()
			return response
		}
		result = fold(result, output.Options)
	}

	if err := d.state.Save(request.Name, result, 0); err != nil {
		logrus.Errorf("Save volume name=%s failed, err: %s", request.Name, err)
		d.exec("delete", toArgs(request.Name, result))
		response.Err = err.Error()
		return response
	}

	return response
}

func (d *RancherStorageDriver) List(request volume.Request) volume.Response {
	response := volume.Response{}
	volumes, err := d.state.List()
	if err != nil {
		response.Err = err.Error()
	} else {
		response.Volumes = volumes
	}

	return response
}

func (d *RancherStorageDriver) Get(request volume.Request) volume.Response {
	response := volume.Response{}
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
	output := &CmdOutput{}
	defer logResponse("remove", request.Name, &response, output)

	_, rVol, err := d.state.Get(request.Name)
	if err == errNoSuchVolume {
		return volume.Response{}
	} else if err != nil {
		response.Err = err.Error()
		return response
	}

	// Docker removal is fake, unless Rancher initiated removal of resource, then we do it.
	if rVol.State == "removing" {
		var err error
		*output, err = d.exec("delete", toArgs(request.Name, getOptions(rVol)))
		if err != nil {
			response.Err = err.Error()
			return response
		}
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

func (d *RancherStorageDriver) doAttach(name, opts string) (*CmdOutput, error) {
	cmdOutput, err := d.exec("attach", opts)
	if err != nil && err != errNotSupported {
		logrus.Errorf("Failed to attach, opts==%s: %v", opts, err)
		return nil, err
	}

	return &cmdOutput, nil
}

func (d *RancherStorageDriver) Attach(request AttachRequest) volume.Response {
	d.mountLock.Lock()
	defer d.mountLock.Unlock()

	logrus.WithFields(logrus.Fields{
		"name": request.Name,
	}).Info("attach.request")

	response := volume.Response{}
	output := &CmdOutput{}
	defer logResponse("attach", request.Name, &response, output)

	_, rVol, err := d.state.Get(request.Name)
	if err != nil {
		response.Err = err.Error()
		return response
	}

	opts := toArgs(request.Name, getOptions(rVol))
	output, err = d.doAttach(request.Name, opts)
	if err != nil {
		response.Err = err.Error()
		return response
	}

	// If SaveOnAttach, update driver Options.
	if d.SaveOnAttach {
		options := getOptions(rVol)
		options["device"] = output.Device
		if err := d.state.Save(request.Name, options, 0); err != nil {
			logrus.Errorf("Save volume name=%s failed, err: %s", request.Name, err)
			response.Err = err.Error()
			return response
		}
	}

	return response
}

func (d *RancherStorageDriver) Mount(request volume.MountRequest) volume.Response {
	d.mountLock.Lock()
	defer d.mountLock.Unlock()

	logrus.WithFields(logrus.Fields{
		"name": request.Name,
	}).Info("mount.request")

	response := volume.Response{}
	output := &CmdOutput{}
	defer logResponse("mount", request.Name, &response, output)

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
	output, err = d.doAttach(request.Name, opts)
	if err != nil && err != errNotSupported {
		logrus.Errorf("Failed to attach %s: %v", request.Name, err)
		response.Err = err.Error()
		return response
	}

	os.MkdirAll(mntDest, 0750)
	*output, err = d.exec("mount", mntDest, output.Device, opts)
	if err != nil {
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
	defer logResponse("unmount", request.Name, &response, &CmdOutput{})

	d.kickGC()
	return response
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

	logrus.Infof("Detaching %s", device)
	if _, err := d.exec("detach", device); err != nil && err != errNotSupported {
		return errors.Wrapf(err, "detach %s", device)
	}

	if _, err := os.Stat(mntDest); err == nil {
		if notmnt, err := d.mounter.IsLikelyNotMountPoint(mntDest); err != nil {
			return errors.Wrap(err, "Lookup mount")
		} else if notmnt {
			if err := os.Remove(mntDest); err != nil {
				return errors.Wrapf(err, "delete %s", mntDest)
			}
		}
	}
	logrus.Infof("Umounting %s done", mntDest)

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

	d.mountMapLock.RLock()
	for src, ids := range d.mountMap {
		if len(ids) == 0 {
			if toCheck[src] {
				toUnmount[src] = true
			}
		}
	}
	for mount := range toCheck {
		if _, ok := d.mountMap[mount]; !ok {
			logrus.Errorf("Mounted but not registered in Docker: %s", mount)
			toUnmount[mount] = true
		}
	}
	d.mountMapLock.RUnlock()

	var lastErr error
	for mnt := range toUnmount {
		if err := d.unmount(mnt); err != nil {
			lastErr = err
			logrus.Errorf("Failed to unmount %s: %v", mnt, err)
		}
	}

	return lastErr
}

func (d *RancherStorageDriver) ListAllVolumes() ([]*volume.Volume, error) {
	vols, err := d.state.client.Volume.List(&client.ListOpts{
		Filters: map[string]interface{}{
			"removed_null":    "true",
			"limit":           "-1",
			"storageDriverId": d.state.driverID,
		},
	})
	if err != nil {
		return nil, err
	}
	result := []*volume.Volume{}
	for _, vol := range vols.Data {
		result = append(result, volToVol(vol))
	}
	return result, nil
}

func (d *RancherStorageDriver) watchContainerEvents() error {
	for {
		reader, err := d.cli.Events(context.Background(), types.EventsOptions{})
		if err != nil {
			logrus.Error(err)
			continue
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
				d.mountMapLock.Lock()
				for _, mountsMap := range d.mountMap {
					delete(mountsMap, event.ID)
				}
				d.mountMapLock.Unlock()
				d.kickGC()
			} else if event.Status == "start" {
				inspect, err := d.cli.ContainerInspect(context.Background(), event.ID)
				if err != nil {
					logrus.Errorf("failed to inspect new created container, err: %v", err)
					continue
				}
				d.mountMapLock.Lock()
				for _, mount := range inspect.Mounts {
					if strings.HasPrefix(mount.Source, d.getMntRoot()) {
						if ids, ok := d.mountMap[mount.Source]; ok {
							ids[event.ID] = struct{}{}
						} else {
							d.mountMap[mount.Source] = map[string]struct{}{}
							d.mountMap[mount.Source][event.ID] = struct{}{}
						}
					}
				}
				d.mountMapLock.Unlock()
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func syncMountMap(d *RancherStorageDriver, cli *dockerClient.Client) {
	for {
		containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
		if err != nil {
			continue
		}
		d.mountMapLock.Lock()
		for _, container := range containers {
			for _, mount := range container.Mounts {
				if strings.HasPrefix(mount.Source, d.getMntRoot()) {
					if ids, ok := d.mountMap[mount.Source]; ok {
						ids[container.ID] = struct{}{}
					} else {
						d.mountMap[mount.Source] = map[string]struct{}{}
						d.mountMap[mount.Source][container.ID] = struct{}{}
					}
				}
			}
		}
		d.mountMapLock.Unlock()
		time.Sleep(time.Minute * 1)
	}
}
