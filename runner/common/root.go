package common

import "github.com/powerpixel/pipelinefox/parser/common"

type PipelineRunner interface {
	RunPipeline(pipeline common.PipelineDescriptor) (string, error)
	RunPipelineJob(job common.PipelineJobDescriptor) (string, error)
}