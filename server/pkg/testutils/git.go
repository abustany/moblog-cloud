package testutils

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

func GitErr(t *testing.T, args ...string) (string, error) {
	gitPath, err := exec.LookPath("git")

	if err != nil {
		t.Fatalf("Cannot find git in PATH")
	}

	var stdoutBuffer bytes.Buffer
	var stderrBuffer bytes.Buffer
	gitCmd := exec.Command(gitPath, args...)
	gitCmd.Env = []string{"GIT_TERMINAL_PROMPT=0"}
	gitCmd.Stdin = nil
	gitCmd.Stdout = &stdoutBuffer
	gitCmd.Stderr = &stderrBuffer

	t.Logf("Running git %v", args)

	if err := gitCmd.Run(); err != nil {
		return "", errors.Errorf("Git command failed. Stderr: %s", stderrBuffer.String())
	}

	return strings.TrimSpace(stdoutBuffer.String()), nil
}

func Git(t *testing.T, args ...string) string {
	stdout, err := GitErr(t, args...)

	if err != nil {
		t.Fatal(err)
	}

	return stdout
}
