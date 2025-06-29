package docker

import (
	"archive/tar"
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
	v1 "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/powerpixel/pipelinefox/logging"
	parserCommon "github.com/powerpixel/pipelinefox/parser/common"
	"github.com/powerpixel/pipelinefox/runner/common"
	"github.com/powerpixel/pipelinefox/shell"
	"golang.org/x/sync/errgroup"
)

const (
	defaultImage = "ubuntu:25.10"
	prefix       = "pipelinefox_"
)

type dockerPipelineRunner struct {
	cli client.APIClient
}

func (d dockerPipelineRunner) RunPipeline(stdout, stderr io.Writer, pipeline parserCommon.PipelineDescriptor) (err error) {
	for stage, jobs := range pipeline.GetStages() {
		err := d.runJobs(stdout, stderr, stage, jobs)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d dockerPipelineRunner) runJobs(stdout, stderr io.Writer, stage string, jobs []parserCommon.PipelineJobDescriptor) error {
	fmt.Printf("Running stage %s\n", stage)
	for _, job := range jobs {
		err := d.RunPipelineJob(stdout, stderr, job)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d dockerPipelineRunner) RunPipelineJob(stdout, stderr io.Writer, job parserCommon.PipelineJobDescriptor) error {
	ctx := context.Background()
	image := d.getImageFromJob(job)

	if err := d.checkImageExistence(ctx, image); err != nil {
		if err := d.pullImage(ctx, image); err != nil {
			return fmt.Errorf("failed to pull image %s: %w", image, err)
		}
	}

	createResp, err := d.createContainerForJob(ctx, job, image)
	if err != nil {
		return err
	}

	defer d.removeContainer(ctx, createResp.ID)

	if err = d.startContainer(ctx, createResp.ID); err != nil {
		return err
	}
	d.waitForContainer(ctx, createResp.ID)

	if err = d.injectScriptIntoContainer(ctx, job, createResp.ID); err != nil {
		return fmt.Errorf("failed to inject script into container: %w", err)
	}

	execResp, err := d.cli.ContainerExecCreate(ctx, createResp.ID, container.ExecOptions{
		Cmd:          []string{"sh", "-c", "/tmp/ppfox-bootstrap.sh"},
		AttachStdout: true,
		AttachStderr: true,
		Tty:          false,
	})

	if err != nil {
		return err
	}

	attachResp, err := d.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{
		Tty: false,
	})

	if err != nil {
		return err
	}
	defer attachResp.Close()

	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()


	g := errgroup.Group{}

	g.Go(func() error {
		defer stdoutWriter.Close()
		defer stderrWriter.Close()

		_, err = stdcopy.StdCopy(stdoutWriter, stderrWriter, attachResp.Reader)

		if err != nil {
			return err
		}
		return nil
	})

	g.Go(func() error {
		scanner := bufio.NewScanner(stdoutReader)
		for scanner.Scan() {
			logging.LogJobOutput(job, scanner.Text(), &stdout)
		}

		if scanner.Err() != nil {
			return scanner.Err()
		}
		return nil
	})

	g.Go(func() error {
		scanner := bufio.NewScanner(stderrReader)
		for scanner.Scan() {
			logging.LogJobError(job, scanner.Text(), &stderr)
		}

		if scanner.Err() != nil {
			return scanner.Err()
		}

		return nil
	})

	return g.Wait()
}

func (d dockerPipelineRunner) injectScriptIntoContainer(ctx context.Context, job parserCommon.PipelineJobDescriptor, containerId string) error {
	scriptBuffer := new(bytes.Buffer)

	err := shell.CreateShellScriptFromCommands(scriptBuffer, job.GetScript())

	if err != nil {
		return err
	}

	payload, err := d.createScriptPayload(ctx, *scriptBuffer)

	if err != nil {
		return err
	}

	return d.cli.CopyToContainer(ctx, containerId, "/tmp", payload, container.CopyToContainerOptions{})
}

func (d dockerPipelineRunner) createScriptPayload(ctx context.Context, scriptBuffer bytes.Buffer) (*bytes.Buffer, error) {
	tarBuffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(tarBuffer)

	header := &tar.Header{
		Name: "ppfox-bootstrap.sh",
		Mode: 0755,
		Size: int64(scriptBuffer.Len()),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		return tarBuffer, err
	}

	if _, err := io.Copy(tarWriter, &scriptBuffer); err != nil {
		return tarBuffer, err
	}

	if err := tarWriter.Close(); err != nil {
		return tarBuffer, err
	}

	return tarBuffer, nil
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
		time.Sleep(100 * time.Millisecond)
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
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
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
