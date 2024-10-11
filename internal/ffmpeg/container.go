package ffmpeg

import (
	"archive/tar"
	"context"
	"fmt"
	"io"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type FFmpegContainer struct {
	cli         *client.Client
	containerID string
	ctx         context.Context
}

func (f *FFmpegContainer) Close() error {
	return f.cli.ContainerRemove(f.ctx, f.containerID, container.RemoveOptions{
		Force: true,
	})
}
func (f *FFmpegContainer) Exec(cmd []string) (int, error) {
	execCreateResponse, err := f.cli.ContainerExecCreate(f.ctx, f.containerID, container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	})
	if err != nil {
		return 0, fmt.Errorf("error ExecCreate: %s", err)
	}
	resp, err := f.cli.ContainerExecAttach(f.ctx, execCreateResponse.ID, container.ExecAttachOptions{})
	if err != nil {
		return 0, fmt.Errorf("error ExecAttach: %s", err)
	}
	defer resp.Close()
	for {
		execInspect, err := f.cli.ContainerExecInspect(f.ctx, execCreateResponse.ID)
		if err != nil {
			return 0, fmt.Errorf("error ExecInspect: %s", err)

		}
		if !execInspect.Running {
			return execInspect.ExitCode, nil
		}
	}
}
func (f *FFmpegContainer) Open(path string) ([]byte, error) {
	reader, _, err := f.cli.CopyFromContainer(f.ctx, f.containerID, path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	tr := tar.NewReader(reader)
	_, err = tr.Next()
	if err != nil {
		return nil, fmt.Errorf("error reading tar header: %s", err)
	}
	data, err := io.ReadAll(tr)
	if err != nil {
		return nil, fmt.Errorf("error reading data from container: %s", err)
	}
	return data, nil
}

func NewFFmpegContainer(image string) (*FFmpegContainer, error) {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, fmt.Errorf("error getting docker client: %s", err)
	}
	containerCreateResponse, err := cli.ContainerCreate(ctx, &container.Config{Image: image, Entrypoint: []string{"sleep", "infinity"}}, &container.HostConfig{AutoRemove: true, NetworkMode: container.NetworkMode("host")}, nil, nil, "tgmon-ffmpeg")
	if err != nil {
		return nil, fmt.Errorf("error creating container: %s", err)
	}
	if err := cli.ContainerStart(ctx, containerCreateResponse.ID, container.StartOptions{}); err != nil {
		return nil, fmt.Errorf("error start container: %s", err)
	}
	return &FFmpegContainer{
		cli:         cli,
		containerID: containerCreateResponse.ID,
		ctx:         ctx,
	}, nil
}
