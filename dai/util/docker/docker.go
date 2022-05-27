package docker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"

	"github.com/PaddlePaddle/PaddleDTX/dai/config"
	"github.com/PaddlePaddle/PaddleDTX/dai/util/logging"
)

const PADDLEFL_CONTAINER_IMAGE = "registry.baidubce.com/paddledtx/paddledtx-paddlefl:1.1.2"
const PADDLEFL_LOCAL_WORKSPACE = "/home/paddlefl/%s/"
const PADDLEFL_CONTAINER_WORKSPACE = "/workspace/%s/"

// ContainerInfo is the information required in creating a container, like the parameters of terminal command 'docker run'
type ContainerInfo struct {
	// image's name,  contains repository name and tag
	Image string
	// container's name
	Name string
	// commands executed one the container created
	Cmd []string
	// container's workspace
	WorkingDir string
	// ports mapping between physical machine and docker container, every string is equal to '-p' in 'docker run'
	Port []string
	// volumes mapping between physical machine and docker container, every string is equal to '-v' in 'docker run'
	Volume []string
}

// CreateAndStartContainer create and start a container
func CreateAndStartContainer(containerInfo *ContainerInfo) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()

	// pull image from a remote registry
	ctx := context.Background()
	reader, err := cli.ImagePull(ctx, containerInfo.Image, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	io.Copy(os.Stdout, reader)

	// container's port
	exports := make(nat.PortSet)
	portMap := make(nat.PortMap)
	for _, value := range containerInfo.Port {
		hostPort, containerPort, err := handleMappingParam(value)
		if err != nil {
			return err
		}
		portBind := nat.PortBinding{HostPort: hostPort}
		port, err := nat.NewPort("tcp", containerPort)
		portMap[port] = []nat.PortBinding{portBind}
		exports[port] = struct{}{}
	}
	// container's hostPath
	mountArr := make([]mount.Mount, 0, len(containerInfo.Volume))
	for _, value := range containerInfo.Volume {
		dir, containerDir, err := handleMappingParam(value)
		if err != nil {
			return err
		}
		mountArr = append(mountArr, mount.Mount{
			Type:   mount.TypeBind,
			Source: dir,
			Target: containerDir,
		})

	}

	// create the container
	hostConfig := &container.HostConfig{
		PortBindings: portMap,
		Mounts:       mountArr,
		NetworkMode:  "host",
	}
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image:        containerInfo.Image,
		Cmd:          containerInfo.Cmd,
		ExposedPorts: exports,
		WorkingDir:   containerInfo.WorkingDir,
	}, hostConfig, nil, containerInfo.Name)
	if err != nil {
		return err
	}
	if err := cli.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return err
	}
	return nil
}

// CheckRunningStatusByContainerName use container's name to check whether the container is running
func CheckRunningStatusByContainerName(name string) (bool, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return false, err
	}
	defer cli.Close()

	ctx := context.Background()
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{})
	for _, container := range containers {
		if name == strings.Trim(container.Names[0], "/") {
			return true, nil
		}
	}
	return false, nil
}

// handleMappingParam
func handleMappingParam(param string) (string, string, error) {
	ret := strings.Split(param, ":")
	fmt.Println(param, ret)
	if len(ret) != 2 {
		return "", "", errors.New("illegal parmas")
	}
	return strings.TrimSpace(ret[0]), strings.TrimSpace(ret[1]), nil
}

func RunCommand(cmd []string, containerName string) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	defer cli.Close()
	ctx := context.Background()

	r, err := cli.ContainerExecCreate(ctx, containerName, types.ExecConfig{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Cmd:          cmd,
		Tty:          false,
	})
	if err != nil {
		return err
	}

	// for stdout log
	attach, err := cli.ContainerExecAttach(ctx, r.ID, types.ExecStartCheck{
		false,
		false,
	})
	if err != nil {
		return err
	}
	defer attach.Close()

	//io.Copy(os.Stdout, attach.Reader)

	config.InitConfig("conf/config.toml")
	logConf := config.GetLogConf()
	logStd, err := logging.InitLog(logConf, "executor.log", true)
	io.Copy(logStd.Writer, attach.Reader)

	err = cli.ContainerExecStart(ctx, r.ID, types.ExecStartCheck{})
	if err != nil {
		return err
	}

	ret, err := cli.ContainerExecInspect(ctx, r.ID)
	if err != nil {
		return err
	}
	if ret.ExitCode != 0 {
		return errors.New("docker exec  failed, exitcode:" + strconv.Itoa(ret.ExitCode))
	}
	return nil
}
