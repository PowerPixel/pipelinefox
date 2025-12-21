package docker

import (
	"bytes"
	"strings"
	"testing"

	parserCommon "github.com/powerpixel/pipelinefox/parser/common"
	"github.com/powerpixel/pipelinefox/runner/common"
	"github.com/powerpixel/pipelinefox/utils"
)

func TestSimpleDockerPipelineExecution(t *testing.T) {
	testCases := []common.RunnerTestCase{
		{
			Title:  "it should run a simple echo command",
			Stages: []string{"build"},
			Jobs: []parserCommon.PipelineJobDescriptor{
				parserCommon.NewPipelineJobDescriptor("test", "build", []string{
					"echo \"hello :)\"",
				}),
			},
			ExpectedOutput: "hello :)\n",
		},
		{
			Title:  "it should run 2 simple echo jobs in the same stage",
			Stages: []string{"build"},
			Jobs: []parserCommon.PipelineJobDescriptor{
				parserCommon.NewPipelineJobDescriptor("test", "build", []string{
					"echo \"hello :)\"",
				}),
				parserCommon.NewPipelineJobDescriptor("test_2", "build", []string{
					"echo \"world ;)\"",
				}),
			},
			ExpectedOutput: `hello :)
world ;)
`,
		},
		{
			Title:  "it should run a simple job with multiple script lines",
			Stages: []string{"build"},
			Jobs: []parserCommon.PipelineJobDescriptor{
				parserCommon.NewPipelineJobDescriptor("test", "build", []string{
					"echo \"hello :)\"",
					"echo \"world ;)\"",
				}),
			},
			ExpectedOutput: `hello :)
world ;)
`,
		},
		{
			Title:  "it should output string on stderr",
			Stages: []string{"build"},
			Jobs: []parserCommon.PipelineJobDescriptor{
				parserCommon.NewPipelineJobDescriptor("test", "build", []string{
					"echo \"error :(\" >> /dev/stderr",
				}),
			},
			ExpectedErrorOutput: "error :(\n",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Title, func(t *testing.T) {
			stdout, stderr := new(bytes.Buffer), new(bytes.Buffer)
			runner := createNewDockerRunner(t)

			pipeline := common.CreateNewPipelineDescriptor(t, testCase.Stages, testCase.Jobs)

			err := runner.RunPipeline(stdout, stderr, pipeline)

			expectNoError(t, err)

			for _, s := range strings.Split(testCase.ExpectedOutput, "\n") {
				utils.ExpectContainString(t, s, stdout.String())
			}

			for _, s := range strings.Split(testCase.ExpectedErrorOutput, "\n") {
				utils.ExpectContainString(t, s, stderr.String())
			}

		})
	}

}

func expectNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("expected pipeline to run without issue but encountered %s", err)
	}
}

func createNewDockerRunner(t testing.TB) common.PipelineRunner {
	t.Helper()
	res, err := NewDockerPipelineRunner()

	if err != nil {
		t.Fatalf("unexpected error during docker runner creation : %s", err.Error())
	}

	return res
}
