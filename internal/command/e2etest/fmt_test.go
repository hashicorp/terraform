// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package e2etest

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/e2e"
)

// Regression test for https://github.com/hashicorp/terraform/issues/39299
func TestFmt_errorWritingToFile(t *testing.T) {
	switch runtime.GOOS {
	case "darwin", "linux":
	default:
		t.Skipf("test requires a Unix shell; `sh` unsupported on %s", runtime.GOOS)
	}

	fixturePath := filepath.Join("testdata", "fmt")
	tf := e2e.NewBinary(t, terraformBin, fixturePath)

	// Assert that main.tf has content before running fmt
	mainPath := filepath.Join(tf.WorkDir(), "main.tf")
	originalContent, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("unexpected error reading test file: %s", err)
	}
	if len(originalContent) == 0 {
		t.Fatal("expected main.tf to contain config, but it is empty")
	}

	// Assert that there's a formatting issue present
	// With `-check`, this is confirmed by error code 3.
	preFmtCmd := tf.Cmd("fmt", "-check", "-no-color")
	err = preFmtCmd.Run()
	if err.Error() != "exit status 3" {
		t.Fatalf("expected exit status 3 error, got: %s", err)
	}

	// The fmt command we're testing ulimit with, which will attempt to write to main.tf
	fmtCmd := tf.Cmd("fmt", "-no-color")
	fmtCmd.Stdin = nil
	fmtCmd.Stdout = &bytes.Buffer{}
	fmtCmd.Stderr = &bytes.Buffer{}

	// But, we need to wrap the command above in a shell command to enforce `ulimit -f 0`.
	// This:
	//   * Causes an error in fmt when writing formatted content to the file.
	//   * Only impacts this command and not the entire test process.
	cmd := exec.Command(
		"/bin/sh", "-c",
		`ulimit -f 0; exec "$@"`,
		"sh", fmtCmd.Path,
	)
	cmd.Args = append(cmd.Args, fmtCmd.Args[1:]...)
	cmd.Dir = fmtCmd.Dir
	cmd.Env = fmtCmd.Env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = fmtCmd.Stdin, fmtCmd.Stdout, fmtCmd.Stderr

	err = cmd.Run()
	if err == nil {
		t.Fatal("expected error when writing to file with ulimit -f 0, but got none")
	}

	stderr := cmd.Stderr.(*bytes.Buffer).String()
	expectErr := "Error: Failed to write main.tf"
	if !strings.Contains(stderr, expectErr) {
		t.Fatalf("expected stderr to contain '%s', but got: %s", expectErr, stderr)
	}

	// Finally, confirm that the error hasn't impacted the original content in main.tf
	latestContent, err := os.ReadFile(mainPath)
	if err != nil {
		t.Fatalf("unexpected error reading test file: %s", err)
	}
	if len(latestContent) == 0 {
		t.Fatalf("expected main.tf to not be empty after error, but got: %s", string(latestContent))
	}
	if !bytes.Equal(originalContent, latestContent) {
		t.Fatalf("expected main.tf content to remain unchanged, but it changed")
	}
}
