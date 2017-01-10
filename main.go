package main

import (
	"context"
	"os"

	"github.com/Sirupsen/logrus"
	dockerClient "github.com/docker/engine-api/client"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/pkg/errors"
	"github.com/rancher/go-rancher/v2"
	"github.com/rancher/kubernetes-agent/healthcheck"
	"github.com/rancher/storage/docker/volumeplugin"
	"github.com/urfave/cli"
)

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
		cli.IntFlag{
			Name:  "healthcheck-port",
			Usage: "listen port for healthchecks",
		},
		cli.StringFlag{
			Name:   "docker-host",
			Value:  "unix:///var/run/docker.sock",
			Usage:  "The DOCKER_HOST to connect to",
			EnvVar: "DOCKER_HOST",
		},
	}
	logrus.Info("Running")
	app.Run(os.Args)
}

func start(c *cli.Context) error {
	logrus.Info("Starting")
	cli, err := dockerClient.NewClient(c.String("docker-host"), "v1.22", nil, nil)
	if err != nil {
		return err
	}

	if _, err := cli.Info(context.Background()); err != nil {
		return err
	}

	opts := &client.ClientOpts{
		Url:       c.String("cattle-url"),
		AccessKey: c.String("cattle-access-key"),
		SecretKey: c.String("cattle-secret-key"),
	}
	client, err := client.NewRancherClient(opts)
	if err != nil {
		return err
	}

	driverName := c.String("driver-name")
	if driverName == "" {
		return errors.New("--driver-name is required")
	}
	d, err := volumeplugin.NewRancherStorageDriver(driverName, client, cli)
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
	if err != nil {
		return err
	}

	logrus.Infof("Starting plugin for %s", driverName)
	h := volume.NewHandler(d)
	if c.Int("healthcheck-port") > 0 {
		go func() {
			err := healthcheck.StartHealthCheck(c.Int("healthcheck-port"))
			logrus.Fatalf("Error while running healthcheck [%v]", err)
		}()
	}
	volumeplugin.ExtendHandler(h, d)
	return h.ServeUnix("root", driverName)
}
