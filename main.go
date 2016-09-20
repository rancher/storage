package main

import (
	"errors"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/kubernetes-agent/healthcheck"
	"github.com/rancher/storage/docker/volumeplugin"
	"github.com/urfave/cli"
)

const healthCheckPort = 10241

var VERSION = "v0.0.0-dev"

func main() {
	app := cli.NewApp()
	app.Name = "storage"
	app.Version = VERSION
	app.Usage = "Magic"
	app.Action = func(c *cli.Context) error {
		if err := start(c); err != nil {
			logrus.Fatal(err)
		}
		return nil
	}
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "driver-name",
			Usage: "The volume driver name",
		},
		cli.StringFlag{
			Name:   "cattle-url",
			Usage:  "The URL endpoint to communicate with cattle server",
			EnvVar: "CATTLE_URL",
		},
		cli.StringFlag{
			Name:   "cattle-access-key",
			Usage:  "The access key required to authenticate with cattle server",
			EnvVar: "CATTLE_ACCESS_KEY",
		},
		cli.StringFlag{
			Name:   "cattle-secret-key",
			Usage:  "The secret key required to authenticate with cattle server",
			EnvVar: "CATTLE_SECRET_KEY",
		},
		cli.IntFlag{
			Name:  "healthcheck-interval",
			Value: 5000,
			Usage: "set the frequency of performing healthchecks",
		},
	}
	logrus.Info("Running")
	app.Run(os.Args)
}

func start(c *cli.Context) error {
	logrus.Info("Starting")
	opts := &client.ClientOpts{
		Url:       c.String("cattle-url"),
		AccessKey: c.String("cattle-access-key"),
		SecretKey: c.String("cattle-secret-key"),
	}
	logrus.Info("Opts %v", opts)
	client, err := client.NewRancherClient(opts)
	if err != nil {
		return err
	}

	driverName := c.String("driver-name")
	if driverName == "" {
		return errors.New("--driver-name is required")
	}
	d := volumeplugin.NewRancherStorageDriver(driverName, client)
	//		DriveName:       driver,
	//		Basedir:         DefaultBasedir,
	//		CreateSupported: true,
	//		Command:         driver,
	//		client:          client,
	//		state: &RancherState{
	//			client: client,
	//		},
	//		mounter: &mount.SafeFormatAndMount{Interface: mount.New(), Runner: exec.New()},
	//		FsType:  DefaultFsType,

	logrus.Infof("Starting plugin for %s", driverName)
	h := volume.NewHandler(d)
	go func() {
		err := healthcheck.StartHealthCheck(healthCheckPort)
		logrus.Fatalf("Error while running healthcheck [%v]", err)
	}()
	return h.ServeUnix("root", driverName)
}
