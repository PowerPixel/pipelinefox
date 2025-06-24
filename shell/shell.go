package shell

import (
	"io"
	"path/filepath"
	"text/template"
)

const templFile = "templates/sh.tmpl"

func CreateShellScriptFromCommands(w io.Writer, script []string) error {

	tmpl, err := template.New(templFile).ParseFiles(templFile)
	if err != nil {
		return err
	}

	err = tmpl.ExecuteTemplate(w, filepath.Base(templFile), script)

	if err != nil {
		return err
	}
	return nil
}
