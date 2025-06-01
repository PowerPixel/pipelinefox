package common

import (
	"errors"
	"fmt"
	"maps"
	"slices"
)

type StageJobMap map[string][]PipelineJobDescriptor

func (s StageJobMap) GetJobs() []PipelineJobDescriptor {
	var result []PipelineJobDescriptor

	for jobs := range maps.Values(s) {
		result = append(result, jobs...)
	}
	return result
}

type CiParser interface {
	ParsePipelineDescriptor(content string) (PipelineDescriptor, error)
}

type PipelineDescriptor struct {
	stages StageJobMap
}

func (p PipelineDescriptor) GetStages() StageJobMap {
	return p.stages
}

type PipelineJobDescriptor struct {
	name   string
	stage string
	script []string
}

func (j PipelineJobDescriptor) GetName() string {
	return j.name 
}

func (j PipelineJobDescriptor) GetStage() string {
	return j.stage
}

func (j PipelineJobDescriptor) GetScript() []string {
	return j.script
}

func NewPipelineDescriptor(stages []string, jobs []PipelineJobDescriptor) (*PipelineDescriptor, error) {
	resultStages := make(StageJobMap)
	
	for _, job := range jobs {
		jobStage := job.GetStage()
		if ! slices.Contains(stages, jobStage) {
			return nil, errors.New(fmt.Sprintf("Unknown stage %s for job %s", jobStage, job.GetName()))
		}
		stage, found := resultStages[jobStage]

		if ! found {
			resultStages[jobStage] = []PipelineJobDescriptor {
				job,
			}
			continue
		}

		resultStages[jobStage] = append(stage, job)
	}
	
	return &PipelineDescriptor{
		resultStages,
	}, nil
}

func NewPipelineJobDescriptor(name string, stage string, script []string) PipelineJobDescriptor {
	return PipelineJobDescriptor{
		name,
		stage,
		script,
	}
}
