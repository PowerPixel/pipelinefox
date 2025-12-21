package logging

import (
	"bytes"
	"testing"

	"github.com/powerpixel/pipelinefox/parser/common"
	"github.com/powerpixel/pipelinefox/utils"
)

const (
	MAX_BUFFER_SIZE = 1024
)

func TestLogJobOutput(t *testing.T) {
	type args struct {
		job common.PipelineJobDescriptor
		log string
	}
	type expected string
	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "it should simply log a hello",
			args: args{
				job: common.NewPipelineJobDescriptor("test", "stage", nil),
				log: "hello",
			},
			expected: "[stage/test] hello\n",
		},
	}

	out := new(bytes.Buffer)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			LogJobOutput(tt.args.job, tt.args.log, out)
			result := out.String()
			utils.ExpectEqualString(t, string(tt.expected), result)

			out.Reset()
		})
	}
}

func TestLogJobError(t *testing.T) {
	type args struct {
		job common.PipelineJobDescriptor
		log string
	}

	type expected string
	tests := []struct {
		name     string
		args     args
		expected expected
	}{
		{
			name: "it should simply log a hello",
			args: args{
				job: common.NewPipelineJobDescriptor("test", "stage", nil),
				log: "hello",
			},
			expected: "\x1b[31m[stage/test] hello\x1b[0m\n",
		},
	}

	out := new(bytes.Buffer)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			LogJobError(tt.args.job, tt.args.log, out)
			result := out.String()
			utils.ExpectEqualString(t, string(tt.expected), result)

			out.Reset()
		})
	}
}
