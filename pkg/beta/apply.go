package beta

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

func Apply(ctx context.Context, cli *client.Client, name string, params map[string]interface{}, connections map[string]interface{}) error {
	return executeStep(ctx, cli, PROVISIONER_ACTION_APPLY, name, params, connections)
}

func executeStep(ctx context.Context, cli *client.Client, action int, name string, params map[string]interface{}, connections map[string]interface{}) error {
	currentUser, err := user.Current()
	if err != nil {
		panic(err)
	}

	workingDir, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	metadata := ProvisioningMetadata{
		Name: name,
		Tags: map[string]string{},
	}
	params["md_metadata"] = metadata

	tempDir, err := os.MkdirTemp("", ".massdriver-stepname")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	paramsPath := filepath.Join(tempDir, "params.json")
	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return err
	}
	os.WriteFile(paramsPath, paramsBytes, 0644)

	connectionsPath := filepath.Join(tempDir, "connections.json")
	connectionsBytes, err := json.Marshal(connections)
	if err != nil {
		return err
	}
	os.WriteFile(connectionsPath, connectionsBytes, 0644)

	containerName := "foobar"

	hostConfig := container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: workingDir,
				Target: "/src/bundles",
			},
			{
				Type:   mount.TypeBind,
				Source: paramsPath,
				Target: "/src/bundles/params.json",
			},
			{
				Type:   mount.TypeBind,
				Source: connectionsPath,
				Target: "/src/bundles/connections.json",
			},
		},
	}

	var provisionerActionEnv string
	if action == PROVISIONER_ACTION_APPLY {
		provisionerActionEnv = "PROVISIONER_ACTION=apply"
	} else if action == PROVISIONER_ACTION_DESTROY {
		provisionerActionEnv = "PROVISIONER_ACTION=destroy"
	} else {
		return errors.New("unsupported action")
	}

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "provisioner-terraform",
		User:  fmt.Sprintf("%s:%s", currentUser.Uid, currentUser.Gid),
		Env:   []string{provisionerActionEnv},
	}, &hostConfig, nil, nil, containerName)
	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		panic(err)
	}

	go func() {
		reader, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{
			ShowStdout: true,
			ShowStderr: true,
			Follow:     true,
			Timestamps: false,
		})
		if err != nil {
			panic(err)
		}
		defer reader.Close()

		stdcopy.StdCopy(os.Stdout, os.Stderr, reader)
	}()

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		if err != nil {
			panic(err)
		}
	case <-statusCh:
	}

	return cli.ContainerRemove(ctx, containerName, container.RemoveOptions{})
}
