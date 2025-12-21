package logging

import (
	"fmt"
	"io"

	"github.com/powerpixel/pipelinefox/parser/common"
)

func LogJobOutput(job common.PipelineJobDescriptor, log string, out io.Writer) {
	out.Write([]byte(
		fmt.Sprintf("[%s/%s] %s\n", job.GetStage(), job.GetName(), log),
	))
}

func LogJobError(job common.PipelineJobDescriptor, log string, out io.Writer) {
	out.Write([]byte(
		fmt.Sprintf("\x1b[31m[%s/%s] %s\x1b[0m\n", job.GetStage(), job.GetName(), log),
	))
}
