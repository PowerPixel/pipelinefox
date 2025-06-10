package docker

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
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

func (d dockerPipelineRunner) RunPipeline(pipeline parserCommon.PipelineDescriptor) (stdout, stderr string, err error) {
	var stdoutSb, stderrSb strings.Builder

	for stage, jobs := range pipeline.GetStages() {
		stdout, stderr, err := d.runJobs(stage, jobs)
		if err != nil {
			return "", "", err
		}
		stdoutSb.WriteString(strings.TrimSpace(stdout))
		stderrSb.WriteString(strings.TrimSpace(stderr))
	}
	return stdoutSb.String(), stderrSb.String(), nil
}

func (d dockerPipelineRunner) runJobs(stage string, jobs []parserCommon.PipelineJobDescriptor) (stdout, stderr string, err error) {
	var stdoutSb, stderrSb strings.Builder
	fmt.Printf("Running stage %s\n", stage)
	for _, job := range jobs {
		stdout, stderr, err := d.RunPipelineJob(job)
		if err != nil {
			return stdout, stderr, err
		}
		stdoutSb.WriteString(stdout)
		stdoutSb.WriteRune('\n')

		stderrSb.WriteString(stderr)
		stderrSb.WriteRune('\n')
	}
	return stdoutSb.String(), stderrSb.String(), nil
}

func (d dockerPipelineRunner) RunPipelineJob(job parserCommon.PipelineJobDescriptor) (stdout string, stderr string, err error) {
	ctx := context.Background()
	image := d.getImageFromJob(job)

	if err := d.checkImageExistence(ctx, image); err != nil {
		d.pullImage(ctx, image)
	}

	createResp, err := d.createContainerForJob(ctx, job, defaultImage)
	if err != nil {
		return "", "", err
	}

	defer d.removeContainer(ctx, createResp.ID)

	if err := d.startContainer(ctx, createResp.ID); err != nil {
		return "", "", err
	}
	d.waitForContainer(ctx, createResp.ID)

	var stdoutSb, stderrSb strings.Builder
	stdoutSb.Reset()
	stderrSb.Reset()

	for _, line := range job.GetScript() {
		execResp, err := d.cli.ContainerExecCreate(ctx, createResp.ID, container.ExecOptions{
			Cmd:          strings.Fields(line),
			AttachStdout: true,
			AttachStderr: true,
			Tty:          false,
		})

		if err != nil {
			return "", "", err
		}

		attachResp, err := d.cli.ContainerExecAttach(ctx, execResp.ID, container.ExecStartOptions{
			Tty: false,
		})

		if err != nil {
			return "", "", err
		}
		defer attachResp.Close()

		var stdoutBuf, stderrBuf bytes.Buffer
		stdoutBufWriter := bufio.NewWriter(&stdoutBuf)
		stderrBufWriter := bufio.NewWriter(&stderrBuf)
		_, err = stdcopy.StdCopy(stdoutBufWriter, stderrBufWriter, attachResp.Reader)
		stdoutBufWriter.Flush()
		stderrBufWriter.Flush()

		if err != nil {
			return "", "", err
		}

		errCh := make(chan error)
		waitCh := make(chan struct{})

		go func() {
			var wg sync.WaitGroup
			wg.Add(2)
			stdoutScanner := bufio.NewScanner(&stdoutBuf)
			go func() {
				defer wg.Done()
				for stdoutScanner.Scan() {
					s := strings.TrimSpace(stdoutScanner.Text())
					if s == "" {
						continue
					}
					if stdoutScanner.Err() != nil {
						errCh <- stdoutScanner.Err()
						return
					}
					stdoutSb.WriteString(s)
				}
				stdoutSb.WriteRune('\n')

			}()

			stderrScanner := bufio.NewScanner(&stderrBuf)
			go func() {
				defer wg.Done()
				for stderrScanner.Scan() {
					s := strings.TrimSpace(stderrScanner.Text())
					if s == "" {
						continue
					}
					if stderrScanner.Err() != nil {
						errCh <- stderrScanner.Err()
						return
					}
					stderrSb.WriteString(s)
				}
				stderrSb.WriteRune('\n')
			}()
			wg.Wait()
			close(waitCh)
		}()
		select {
		case <-waitCh:
		case err := <-errCh:
			return "", "", err
		}
		close(errCh)
	}
	return strings.TrimSpace(stdoutSb.String()), strings.TrimSpace(stderrSb.String()), nil
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
