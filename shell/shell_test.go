package shell

import (
	"bytes"
	"strings"
	"testing"

	"github.com/powerpixel/pipelinefox/utils"
)

type TestCaseShell struct {
	title                  string
	err                    error
	expectedScriptProduced string
	shellInput             []string
}

func TestSimpleShellScriptCreation(t *testing.T) {
	testCases := []TestCaseShell{
		{
			title: "Simple shell script",
			shellInput: []string{
				"echo 'Hello World!'",
			},
			err:                    nil,
			expectedScriptProduced: utils.ReadTestFile(t, "testdata/simple.sh"),
		},
	}
	var outputBuff bytes.Buffer

	for _, tc := range testCases {
		outputBuff.Reset()
		t.Run(tc.title, func(t *testing.T) {
			err := CreateShellScriptFromCommands(&outputBuff, tc.shellInput)

			if err != nil {
				t.Fatalf("unexpected err: %s", err.Error())
			}
			assertOutputEquality(t, tc.expectedScriptProduced, outputBuff.String())
		})
	}
}

func assertOutputEquality(t *testing.T, expected, actual string) {
	t.Helper()

	expected = trimString(expected)
	actual = trimString(actual)

	if expected != actual {
		t.Fatalf("expected output\n-----\n%s\n-----\nactual\n-----\n%s\n-----\n", expected, actual)
	}
}

func trimString(str string) string {
	return strings.Trim(strings.Trim(str, " "), "\n")
}
