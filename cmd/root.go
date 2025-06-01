package cmd

import (
	"bytes"
	"fmt"
	"os"

	"github.com/powerpixel/pipelinefox/cmd/detector"
	"github.com/powerpixel/pipelinefox/parser/gitlab"
	"github.com/powerpixel/pipelinefox/runner/docker"
	"github.com/spf13/cobra"
)

var scanPath string

var rootCmd = &cobra.Command{
	Use:   "pipelinefox",
	Short: "PipelineFox is a local CI runner to locally test pipelines",
	Long:  "PipelineFox is a local CI runner to locally test pipelines. It aims to be compatible with multiple CI formats.",
	Run: func(cmd *cobra.Command, args []string) {
		file, err := detector.CheckGitlabCi(scanPath)
		if err != nil {
			panic(err)
		}

		if file == nil {
			fmt.Printf("No CI file was found in %v :( \n", scanPath)
			return
		}

		fmt.Printf("Found CI file : %v\n", file.Name())

		ciParser := gitlab.NewGitlabPipelineParser()

		buf := new(bytes.Buffer)
		buf.ReadFrom(file)
		pipeline, err := ciParser.ParsePipelineDescriptor(buf.Bytes())

		if err != nil {
			panic(err)
		}

		runner, err := docker.NewDockerPipelineRunner()

		if err != nil {
			fmt.Printf("encountered unexpected error when trying to create a pipeline runner : %s", err.Error())
			os.Exit(1)
		}

		runner.RunPipeline(*pipeline)
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&scanPath, "path", "", "Path for execution context. Pipelinefox will look for CI declarations here.")
}

func initConfig() {
	if scanPath == "" {
		var err error
		scanPath, err = os.Getwd()
		if err != nil {
			panic(err)
		}
	}

	fmt.Printf("Running Pipelinefox in directory %s \n", scanPath)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Oops. An error occured while executing Pipelinefox '%s'\n", err)
		os.Exit(1)
	}
}
