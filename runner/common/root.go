package common

import "github.com/powerpixel/pipelinefox/parser/common"

type PipelineRunner interface {
	RunPipeline(pipeline common.PipelineDescriptor) (stdout, stderr string, err error)
	RunPipelineJob(job common.PipelineJobDescriptor) (stdout, stderr string, err error)
}
