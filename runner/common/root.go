package common

import (
	"io"

	"github.com/powerpixel/pipelinefox/parser/common"
)

type PipelineRunner interface {
	RunPipeline(stdout, stderr io.Writer, pipeline common.PipelineDescriptor) error
	RunPipelineJob(stdout, stderr io.Writer, job common.PipelineJobDescriptor) error
}
