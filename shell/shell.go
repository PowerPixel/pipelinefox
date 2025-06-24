package shell

import (
	_ "embed"
	"io"
	"text/template"
)

const templFile = "sh"

//go:embed templates/sh.tmpl
var shTempl string

func CreateShellScriptFromCommands(w io.Writer, script []string) error {

	tmpl, err := template.New(templFile).Parse(shTempl)
	if err != nil {
		return err
	}

	err = tmpl.ExecuteTemplate(w, templFile, script)

	if err != nil {
		return err
	}
	return nil
}
