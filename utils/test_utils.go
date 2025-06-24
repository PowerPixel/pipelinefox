package utils

import (
	"os"
	"testing"
)

func ReadTestFile(t testing.TB, file string) string {
	t.Helper()

	f, err := os.ReadFile(file)

	if err != nil {
		t.Fatalf("could not open test file : %s", file)
	}

	return string(f)
}
