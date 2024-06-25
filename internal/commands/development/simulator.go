package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	dockerContainer "github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

type SimulatorClient interface {
}

func StartDevelopmentSimulator() error {
	cli, err := dockerClient.NewClientWithOpts(dockerClient.FromEnv, dockerClient.WithVersion("1.41"))

	if err != nil {
		return errors.New("docker Engine API is not installed. to install it go to https://docs.docker.com/get-docker/ and follow the instructions")
	}

	hostBinding := nat.PortBinding{
		HostIP:   "0.0.0.0",
		HostPort: "6006",
	}

	containerPort, err := nat.NewPort("tcp", "6006")

	if err != nil {
		return err
	}

	portBinding := nat.PortMap{containerPort: []nat.PortBinding{hostBinding}}

	cwd, err := filepath.Abs("./")

	if err != nil {
		return err
	}

	container, err := cli.ContainerCreate(
		context.Background(),
		&dockerContainer.Config{
			Image:        "massdrivercloud/massdriver-bundle-preview",
			ExposedPorts: nat.PortSet{"6006/tcp": struct{}{}},
		},
		&dockerContainer.HostConfig{
			PortBindings: portBinding,
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: cwd,
					Target: "/app/bundle",
				},
			},
		},
		nil,
		nil,
		"")

	if err != nil {
		return err
	}

	err = cli.ContainerStart(context.Background(), container.ID, dockerContainer.StartOptions{})

	if err != nil {
		return err
	}

	logstream, err := cli.ContainerLogs(context.Background(), container.ID, dockerContainer.LogsOptions{
		Follow:     true,
		ShowStdout: true,
		ShowStderr: true,
	})

	if err != nil {
		return err
	}

	defer logstream.Close()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logstream.Close()
		err = cli.ContainerKill(context.Background(), container.ID, "SIGTERM")
		if err != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}()

	err = printDockerOutput(logstream)

	if err != nil {
		return err
	}

	return nil
}

func printDockerOutput(rd io.Reader) error {
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
		lastLine = scanner.Text()
	}

	errLine := &ErrorLine{}

	val := json.Unmarshal([]byte(lastLine), errLine)

	fmt.Println(val)

	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}
