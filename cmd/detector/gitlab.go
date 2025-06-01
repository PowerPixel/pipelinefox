package detector

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

const GitlabCiFilename = ".gitlab-ci.yml"

func CheckGitlabCi(path string) (*os.File, error) {
	var ciFile *os.File
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		if d.Name() == GitlabCiFilename {
			var err error
			ciFile, err = os.Open(path)
			if err != nil {
				return err
			}
			return io.EOF
		}

		return nil
	})

	if err != nil && err != io.EOF {
		return nil, err
	}
	
	return ciFile, nil
}
