package testutils

import (
	"io/ioutil"
	"testing"
)

func TempDir(t *testing.T, name string) string {
	path, err := ioutil.TempDir("", name)

	if err != nil {
		t.Fatalf("Error while temp directory %s: %s", name, err)
	}

	return path
}
