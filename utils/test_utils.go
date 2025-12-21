package utils

import (
	"os"
	"strings"
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

func ExpectEqualString(t *testing.T, expected, actual string) {
	t.Helper()
	if expected != actual {
		t.Fatalf("expected output %s (len %d), got %s (len %d)",
			expected, len(expected),
			actual, len(actual),
		)
	}
}

func ExpectContainString(t *testing.T, expected, actual string) {
	t.Helper()
	if !strings.Contains(actual, expected) {
		t.Fatalf("expected output %s to contain %s",
			actual,
			expected,
		)
	}
}
