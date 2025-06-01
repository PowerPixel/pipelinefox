package common

import (
	"testing"

	"github.com/powerpixel/pipelinefox/parser/common"
	parserCommon "github.com/powerpixel/pipelinefox/parser/common"
)

type RunnerTestCase struct {
	Title          string
	Stages         []string
	Jobs           []common.PipelineJobDescriptor
	ExpectedOutput string
}

func CreateNewPipelineDescriptor(t testing.TB, stages []string, jobs []parserCommon.PipelineJobDescriptor) parserCommon.PipelineDescriptor {
	res, err := parserCommon.NewPipelineDescriptor(stages, jobs)

	if err != nil {
		t.Fatalf("unexpected error during creation of pipeline descriptor : %s", err.Error())
	}

	return *res
}
