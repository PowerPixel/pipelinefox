package docker

import (
	"testing"

	parserCommon "github.com/powerpixel/pipelinefox/parser/common"
	"github.com/powerpixel/pipelinefox/runner/common"
)

func TestSimpleDockerPipelineExecution(t *testing.T) {
	testCases := []common.RunnerTestCase{
		{
			Title:  "it should run a simple echo command",
			Stages: []string{"build"},
			Jobs: []parserCommon.PipelineJobDescriptor{
				parserCommon.NewPipelineJobDescriptor("test", "build", []string{
					"echo hello :)",
				}),
			},
			ExpectedOutput: "hello :)",
		},
		{
			Title:  "it should run 2 simple echo jobs in the same stage",
			Stages: []string{"build"},
			Jobs: []parserCommon.PipelineJobDescriptor{
				parserCommon.NewPipelineJobDescriptor("test", "build", []string{
					"echo hello :)",
				}),
				parserCommon.NewPipelineJobDescriptor("test_2", "build", []string{
					"echo world ;)",
				}),
			},
			ExpectedOutput: `hello :)
world ;)`,
		},
		{
			Title:  "it should run a simple job with multiple script lines",
			Stages: []string{"build"},
			Jobs: []parserCommon.PipelineJobDescriptor{
				parserCommon.NewPipelineJobDescriptor("test", "build", []string{
					"echo hello :)",
					"echo world ;)",
				}),
			},
			ExpectedOutput: `hello :)
world ;)`,
		},
		{
			Title:  "it should output string on stderr",
			Stages: []string{"build"},
			Jobs: []parserCommon.PipelineJobDescriptor{
				parserCommon.NewPipelineJobDescriptor("test", "build", []string{
					"echo error :( >> /dev/stderr",
				}),
			},
			ExpectedErrorOutput: "error :(",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Title, func(t *testing.T) {
			runner := createNewDockerRunner(t)

			pipeline := common.CreateNewPipelineDescriptor(t, testCase.Stages, testCase.Jobs)

			stdout, stderr, err := runner.RunPipeline(pipeline)

			expectNoError(t, err)
			expectEqualString(t, testCase.ExpectedOutput, stdout)
			expectEqualString(t, testCase.ExpectedErrorOutput, stderr)
		})
	}

}

func expectEqualString(t *testing.T, expected, actual string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected output %s (len %d), got %s (len %d)",
			expected, len(expected),
			actual, len(actual),
		)
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
