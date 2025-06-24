package gitlab

import (
	"reflect"
	"testing"

	"github.com/powerpixel/pipelinefox/parser/common"
	"github.com/powerpixel/pipelinefox/utils"
)

type ParserTestCase struct {
	TestName    string
	YAMLContent string
	Expected    common.PipelineDescriptor
}

func TestParseSimpleYamlGitlabCi(t *testing.T) {
	cases := []ParserTestCase{
		{
			TestName:    "It parses a simple gitlab file correctly",
			YAMLContent: utils.ReadTestFile(t, "testdata/simple.yaml"),
			Expected: createNewPipelineDescriptor(t, []string{
				"build",
			},
				[]common.PipelineJobDescriptor{
					common.NewPipelineJobDescriptor(
						"build_app",
						"build",
						[]string{"echo \"I'm building!\""},
					),
				}),
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.TestName, func(t *testing.T) {
			parser := NewGitlabPipelineParser()
			got, err := parser.ParsePipelineDescriptor([]byte(testCase.YAMLContent))
			if err != nil {
				t.Fatalf("parser returned an error but was not supposed to : %v", err)
			}

			assertPipelineDescriptor(t, *got, testCase.Expected)
		})
	}
}

func createNewPipelineDescriptor(t testing.TB, stages []string, jobs []common.PipelineJobDescriptor) common.PipelineDescriptor {
	t.Helper()
	res, err := common.NewPipelineDescriptor(stages, jobs)

	if err != nil {
		t.Fatalf("unexpected error during pipeline descriptor initialization : %s", err.Error())
	}

	return *res
}

func assertPipelineDescriptor(t testing.TB, got, want common.PipelineDescriptor) {
	t.Helper()

	gotStage := got.GetStages()
	wantStage := want.GetStages()

	if !reflect.DeepEqual(gotStage, wantStage) {
		t.Fatalf("stages mismatch, got %v want %v", gotStage, wantStage)
	}

	if !reflect.DeepEqual(gotStage.GetJobs(), wantStage.GetJobs()) {
		t.Fatalf("jobs mismatch, got %v want %v", gotStage.GetJobs(), wantStage.GetJobs())
	}
}
