package runner

import (
	"github.com/powerpixel/pipelinefox/parser/common"
)

type PipelineRunner interface {
	RunPipeline(common.PipelineDescriptor) error
	RunJob(common.PipelineJobDescriptor) error
}