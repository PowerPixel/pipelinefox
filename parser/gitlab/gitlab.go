package gitlab

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"

	"github.com/powerpixel/pipelinefox/parser/common"

	"github.com/antchfx/jsonquery"
	"github.com/antchfx/xpath"
	"sigs.k8s.io/yaml"
)

var UnknownScriptObjectErr = errors.New("the script tag in the yaml descriptor is neither a string or an array of string, this is not handled	")

const (
	stageQueryTemplate = "//*[stage='%v']"
)

type GitlabPipelineDescriptor struct {
	Stages []string `json:"stages"`
}

type GitlabPipelineParser struct{}

func NewGitlabPipelineParser() GitlabPipelineParser {
	return GitlabPipelineParser{}
}

func (p *GitlabPipelineParser) ParsePipelineDescriptor(content []byte) (*common.PipelineDescriptor, error) {

	json, err := yaml.YAMLToJSON(content)
	if err != nil {
		return nil, err
	}

	doc, err := jsonquery.Parse(bytes.NewReader(json))
	if err != nil {
		return nil, err
	}

	parsedStages, err := parseStages(doc)
	if err != nil {
		panic(err)
	}

	parsedJobs, err := parseJobs(parsedStages, doc)
	if err != nil {
		panic(err)
	}

	descriptor, err := common.NewPipelineDescriptor(
		parsedStages,
		parsedJobs,
	)

	if err != nil {
		return nil, err
	}

	return descriptor, nil
}

func parseStages(root *jsonquery.Node) ([]string, error) {

	stages := jsonquery.FindOne(root, "stages")

	parsedStages := make([]string, 0)

	if val, ok := stages.Value().([]interface{}); ok {
		for _, stage := range val {
			switch stageType := stage.(type) {
			case string:
				fmt.Printf("Discovered stage %v\n", stage)
				parsedStages = append(parsedStages, stageType)
			default:
				fmt.Printf("Skipping uknown job type : %v\n", stage)
			}
		}
	}

	return parsedStages, nil
}

func parseJobs(stages []string, root *jsonquery.Node) ([]common.PipelineJobDescriptor, error) {

	parsedJobs := make([]common.PipelineJobDescriptor, 0)

	for _, stage := range stages {

		jobNodes := jsonquery.QuerySelectorAll(root, xpath.MustCompile(fmt.Sprintf(stageQueryTemplate, stage)))

		for _, stageNode := range jobNodes {
			parsedJob, err := parseJob(stageNode, stage)

			if err != nil {
				return nil, err
			}

			parsedJobs = append(parsedJobs, *parsedJob)
		}
	}

	return parsedJobs, nil
}

func parseJob(node *jsonquery.Node, stage string) (*common.PipelineJobDescriptor, error) {
	fmt.Printf("Parsing job %v\n", node.Data)

	scriptNode := jsonquery.FindOne(node, "script")

	script, err := parseScript(scriptNode)
	if err != nil {
		return nil, err
	}

	parsedJob := common.NewPipelineJobDescriptor(node.Data, stage, script)
	return &parsedJob, nil
}

func parseScript(node *jsonquery.Node) ([]string, error) {
	switch node.Value().(type) {
	case string:
		return []string{node.Value().(string)}, nil
	case []any:
		source := node.Value().([]any)
		// Try to convert it to a []string
		r := make([]string, len(source))

		for _, e := range source {
			switch e := e.(type) {
			case string:
				r = append(r, e)
			default:
				return nil, UnknownScriptObjectErr
			}
		}
		return r, nil
	default:
		fmt.Printf("Node of type %v is unknown...; Value is %v \n", reflect.TypeOf(node.Value()), node.Value())
		return nil, UnknownScriptObjectErr
	}

}
