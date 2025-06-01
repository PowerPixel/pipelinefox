package docker

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	parserCommon "github.com/powerpixel/pipelinefox/parser/common"
	"github.com/powerpixel/pipelinefox/runner/common"
)

const (
	defaultImage = "ubuntu:25.10"
	prefix       = "pipelinefox_"
)

type dockerPipelineRunner struct {
	cli client.APIClient
}

func (d dockerPipelineRunner) RunPipeline(pipeline parserCommon.PipelineDescriptor) (string, error) {
	var sb strings.Builder

	for stage, jobs := range pipeline.GetStages() {
		res, err := d.runJobs(stage, jobs)
		if err != nil {
			return "", err
		}
		sb.WriteString(strings.TrimSpace(res))
	}
	return sb.String(), nil
}

func (d dockerPipelineRunner) runJobs(stage string, jobs []parserCommon.PipelineJobDescriptor) (string, error) {
	var sb strings.Builder
	fmt.Printf("Running stage %s\n", stage)
	for _, job := range jobs {
		output, err := d.RunPipelineJob(job)
		if err != nil {
			return output, err
		}
		sb.WriteString(output)
	}
	return sb.String(), nil
}

func (d dockerPipelineRunner) RunPipelineJob(job parserCommon.PipelineJobDescriptor) (string, error) {
	ctx := context.Background()
	image := d.getImageFromJob(job)

	if err := d.checkImageExistence(ctx, image); err != nil {
		d.pullImage(ctx, image)
	}

	createResp, err := d.createContainerForJob(ctx, job, defaultImage)
	if err != nil {
		return "", err
	}

	defer d.removeContainer(ctx, createResp.ID)

	if err := d.startContainer(ctx, createResp.ID); err != nil {
		return "", err
	}
	d.waitForContainer(ctx, createResp.ID)

	var sb strings.Builder
	sb.Reset()

	for _, line := range job.GetScript() {
		execResp, err := d.cli.ContainerExecCreate(ctx, createResp.ID, container.ExecOptions{
			Cmd:          strings.Fields(line),
			AttachStdout: true,
			AttachStderr: true,
			Tty:          false,
		})

		if err != nil {
			return "", err
		}

		attachResp, err := d.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{
			Tty: false,
		})

		if err != nil {
			return "", err
		}
		defer attachResp.Close()

		var buf bytes.Buffer
		bufWriter := bufio.NewWriter(&buf)
		_, err = stdcopy.StdCopy(bufWriter, bufWriter, attachResp.Reader)
		bufWriter.Flush()

		if err != nil {
			return "", err
		}

		stdoutScanner := bufio.NewScanner(&buf)

		for stdoutScanner.Scan() {
			s := strings.TrimSpace(stdoutScanner.Text())
			if s == "" {
				continue
			}
			if stdoutScanner.Err() != nil {
				return "", stdoutScanner.Err()
			}
			sb.WriteString(s)
		}
	}
	return strings.TrimSpace(sb.String()), nil
}

func (d dockerPipelineRunner) startContainer(ctx context.Context, containerID string) error {
	if err := d.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return err
	}

	if _, err := d.cli.ContainerInspect(ctx, containerID); err != nil {
		return err
	}

	return nil
}

func (d dockerPipelineRunner) createContainerForJob(ctx context.Context, job parserCommon.PipelineJobDescriptor, image string) (*container.CreateResponse, error) {
	createResp, err := d.cli.ContainerCreate(
		ctx,
		&container.Config{
			Image:       image,
			AttachStdin: true,
			Tty:         false,
			Cmd:         []string{"tail", "-f", "/dev/null"},
			OpenStdin:   true,
		},
		&container.HostConfig{},
		&network.NetworkingConfig{},
		v1.DescriptorEmptyJSON.Platform,
		prefix+job.GetName(),
	)

	if err != nil {
		return nil, err
	}

	for warning := range createResp.Warnings {
		fmt.Print(warning)
	}

	return &createResp, nil
}

func (d dockerPipelineRunner) removeContainer(ctx context.Context, containerId string) {
	if containerId == "" {
		return
	}
	fmt.Printf("Deleting container %s...\n", containerId)
	err := d.cli.ContainerRemove(ctx, containerId, container.RemoveOptions{Force: true, RemoveVolumes: true})
	if err != nil {
		panic(fmt.Sprintf("Could not delete container %s : %s\n", containerId, err.Error()))
	}
}

func (d dockerPipelineRunner) waitForContainer(ctx context.Context, id string) {
	inspectResp, err := d.cli.ContainerInspect(ctx, id)
	for err != nil && !inspectResp.State.Running {
		time.Sleep(500 * time.Millisecond)
		inspectResp, err = d.cli.ContainerInspect(ctx, id)
	}
}

func (d dockerPipelineRunner) checkImageExistence(ctx context.Context, image string) error {
	_, err := d.cli.ImageInspect(ctx, image)
	return err
}

func (d dockerPipelineRunner) pullImage(ctx context.Context, img string) error {
	io, err := d.cli.ImagePull(ctx, img, image.PullOptions{})

	if err != nil {
		return err
	}
	defer io.Close()
	scanner := bufio.NewScanner(io)

	for scanner.Scan() {
		if scanner.Err() != nil {
			fmt.Print(scanner.Text())
			return scanner.Err()
		}
	}
	return nil
}

func (d dockerPipelineRunner) getImageFromJob(_ parserCommon.PipelineJobDescriptor) string {
	// TODO : add fetching image from job descriptor
	return defaultImage
}

func NewDockerPipelineRunner() (common.PipelineRunner, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	return dockerPipelineRunner{
		cli,
	}, checkDockerExistence(cli)
}

func checkDockerExistence(cli *client.Client) error {
	_, err := cli.Info(context.Background())
	return err
}
