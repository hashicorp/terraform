// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package e2e

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/plans/planfile"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/states/statefile"
)

// TestExperimentFlag is the name of the environment variable that
// can be set to built a terraform binary with experimental features enabled.
// Any value besides "false" or an empty string will enable the feature.
var TestExperimentFlag = "TF_TEST_EXPERIMENTS"

// Type binary represents the combination of a compiled binary
// and a temporary working directory to run it in.
type binary struct {
	binPath string
	workDir string
	env     []string
}

// NewBinary prepares a temporary directory containing the files from the
// given fixture and returns an instance of type binary that can run
// the generated binary in that directory.
//
// If the temporary directory cannot be created, a fixture of the given name
// cannot be found, or if an error occurs while _copying_ the fixture files,
// this function will panic. Tests should be written to assume that this
// function always succeeds.
func NewBinary(t *testing.T, binaryPath, workingDir string) *binary {
	tmpDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		panic(err)
	}

	// For our purposes here we do a very simplistic file copy that doesn't
	// attempt to preserve file permissions, attributes, alternate data
	// streams, etc. Since we only have to deal with our own fixtures in
	// the testdata subdir, we know we don't need to deal with anything
	// of this nature.
	err = filepath.Walk(workingDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == workingDir {
			// nothing to do at the root
			return nil
		}

		if filepath.Base(path) == ".exists" {
			// We use this file just to let git know the "empty" fixture
			// exists. It is not used by any test.
			return nil
		}

		srcFn := path

		path, err = filepath.Rel(workingDir, path)
		if err != nil {
			return err
		}

		dstFn := filepath.Join(tmpDir, path)

		if info.IsDir() {
			return os.Mkdir(dstFn, os.ModePerm)
		}

		src, err := os.Open(srcFn)
		if err != nil {
			return err
		}
		dst, err := os.OpenFile(dstFn, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.ModePerm)
		if err != nil {
			return err
		}

		_, err = io.Copy(dst, src)
		if err != nil {
			return err
		}

		if err := src.Close(); err != nil {
			return err
		}
		if err := dst.Close(); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		panic(err)
	}

	return &binary{
		binPath: binaryPath,
		workDir: tmpDir,
	}
}

// AddEnv appends an entry to the environment variable table passed to any
// commands subsequently run.
func (b *binary) AddEnv(entry string) {
	b.env = append(b.env, entry)
}

// RemoveEnv removes an entry from the environment variable table passed to any
// commands subsequently run.
func (b *binary) RemoveEnv(name string) {
	for i, e := range b.env {
		if strings.HasPrefix(e, name+"=") {
			b.env = append(b.env[:i], b.env[i+1:]...)
			break
		}
	}
}

// IsolateLocalProviderEnv configures the binary's environment so that
// Terraform's "implicit" local provider search (see
// implicitProviderSource in provider_source.go) cannot reach any
// directory that exists outside of this test's temporary tree.
//
// Without this isolation, the e2e tests that exercise local-only provider
// installation can be perturbed by host-machine state: a real
// $HOME/.terraform.d/plugins, an XDG-located plugins directory, or a
// macOS ~/Library/Application Support/io.terraform/plugins entry on the
// developer's workstation will be discovered by the test binary and
// participate in the install plan. Historically that has caused
// confusing "Failed to query available provider packages" failures and
// other intermittent breakage; see GH-37501 for context.
//
// We isolate by pointing every environment variable consulted by
// terraform's CLI-config and provider-discovery code paths at a fresh,
// empty per-test directory:
//
//   - HOME (Linux/macOS): controls cliconfig.ConfigDir() (i.e.
//     ~/.terraformrc and ~/.terraform.d/plugins) and macOS userdirs
//     (~/Library/Application Support/io.terraform).
//   - XDG_* (Linux): consulted by go-userdirs to compute the XDG data
//     and config search paths.
//   - APPDATA / LOCALAPPDATA (Windows): consulted by go-userdirs for
//     the Windows known-folder hierarchy.
//
// In addition, callers usually combine this with TF_CLI_CONFIG_FILE
// pointing at a blank .terraformrc to neutralise the explicit CLI
// configuration mechanism. Tests that previously did that themselves
// can keep doing so; this method only addresses the implicit
// discovery layer that TF_CLI_CONFIG_FILE does NOT cover.
//
// The returned path is the empty isolated home directory, so callers
// that want to drop a custom file under it (for example a synthetic
// .terraformrc) can do so. The directory is registered with t.TempDir
// machinery so it's automatically cleaned up when the test ends.
func (b *binary) IsolateLocalProviderEnv(t testing.TB) string {
	t.Helper()
	isolated := t.TempDir()

	// HOME is consulted by cliconfig.ConfigDir on Linux/macOS for
	// ~/.terraformrc and ~/.terraform.d/plugins, and by go-userdirs's
	// macOS backend for ~/Library/Application Support/io.terraform.
	b.AddEnv("HOME=" + isolated)

	// go-userdirs's XDG (Linux) backend prefers these env vars over
	// HOME-derived defaults; if any of them point at a real directory
	// on the developer's machine the test will inherit those plugin
	// paths. We pin them to empty strings so the library falls back to
	// XDG-spec defaults under our isolated $HOME (which we know is
	// empty), matching the no-XDG-set unit-test scenario in
	// go-userdirs/userdirs/app_unix_test.go.
	for _, name := range []string{
		"XDG_DATA_HOME",
		"XDG_DATA_DIRS",
		"XDG_CONFIG_HOME",
		"XDG_CONFIG_DIRS",
		"XDG_CACHE_HOME",
	} {
		b.AddEnv(name + "=")
	}

	// Windows known-folder vars consulted by go-userdirs's windows
	// backend. The test suite is documented as Linux/macOS only (see
	// CONTRIBUTING.md) but we set these defensively so that anyone
	// running these tests on Windows in the future gets the same
	// isolation.
	b.AddEnv("APPDATA=" + isolated)
	b.AddEnv("LOCALAPPDATA=" + isolated)

	return isolated
}

// Cmd returns an exec.Cmd pre-configured to run the generated Terraform
// binary with the given arguments in the temporary working directory.
//
// The returned object can be mutated by the caller to customize how the
// process will be run, before calling Run.
func (b *binary) Cmd(args ...string) *exec.Cmd {
	cmd := exec.Command(b.binPath, args...)
	cmd.Dir = b.workDir
	cmd.Env = os.Environ()

	// Disable checkpoint since we don't want to harass that service when
	// our tests run. (This does, of course, mean we can't actually do
	// end-to-end testing of our Checkpoint interactions.)
	cmd.Env = append(cmd.Env, "CHECKPOINT_DISABLE=1")

	cmd.Env = append(cmd.Env, b.env...)

	return cmd
}

// Run executes the generated Terraform binary with the given arguments
// and returns the bytes that it wrote to both stdout and stderr.
//
// This is a simple way to run Terraform for non-interactive commands
// that don't need any special environment variables. For more complex
// situations, use Cmd and customize the command before running it.
func (b *binary) Run(args ...string) (stdout, stderr string, err error) {
	cmd := b.Cmd(args...)
	cmd.Stdin = nil
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}
	err = cmd.Run()
	stdout = cmd.Stdout.(*bytes.Buffer).String()
	stderr = cmd.Stderr.(*bytes.Buffer).String()
	return
}

// Path returns a file path within the temporary working directory by
// appending the given arguments as path segments.
func (b *binary) Path(parts ...string) string {
	args := make([]string, 0, len(parts)+1)
	args = append(args, b.workDir)
	args = append(args, parts...)
	return filepath.Join(args...)
}

// OpenFile is a helper for easily opening a file from the working directory
// for reading.
func (b *binary) OpenFile(path ...string) (*os.File, error) {
	flatPath := b.Path(path...)
	return os.Open(flatPath)
}

// ReadFile is a helper for easily reading a whole file from the working
// directory.
func (b *binary) ReadFile(path ...string) ([]byte, error) {
	flatPath := b.Path(path...)
	return os.ReadFile(flatPath)
}

// FileExists is a helper for easily testing whether a particular file
// exists in the working directory.
func (b *binary) FileExists(path ...string) bool {
	flatPath := b.Path(path...)
	_, err := os.Stat(flatPath)
	return !os.IsNotExist(err)
}

// LocalState is a helper for easily reading the local backend's state file
// terraform.tfstate from the working directory.
func (b *binary) LocalState() (*states.State, error) {
	return b.StateFromFile("terraform.tfstate")
}

// StateFromFile is a helper for easily reading a state snapshot from a file
// on disk relative to the working directory.
func (b *binary) StateFromFile(filename string) (*states.State, error) {
	f, err := b.OpenFile(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stateFile, err := statefile.Read(f)
	if err != nil {
		return nil, fmt.Errorf("Error reading statefile: %s", err)
	}
	return stateFile.State, nil
}

// Plan is a helper for easily reading a plan file from the working directory.
func (b *binary) Plan(path string) (*plans.Plan, error) {
	path = b.Path(path)
	pr, err := planfile.Open(path)
	if err != nil {
		return nil, err
	}
	defer pr.Close()
	plan, err := pr.ReadPlan()
	if err != nil {
		return nil, err
	}
	return plan, nil
}

// SetLocalState is a helper for easily writing to the file the local backend
// uses for state in the working directory. This does not go through the
// actual local backend code, so processing such as management of serials
// does not apply and the given state will simply be written verbatim.
func (b *binary) SetLocalState(state *states.State) error {
	path := b.Path("terraform.tfstate")
	f, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create temporary state file %s: %s", path, err)
	}
	defer f.Close()

	sf := &statefile.File{
		Serial:  0,
		Lineage: "fake-for-testing",
		State:   state,
	}
	return statefile.Write(sf, f)
}

func GoBuild(pkgPath, tmpPrefix string) string {
	dir, prefix := filepath.Split(tmpPrefix)
	tmpFile, err := os.CreateTemp(dir, prefix)
	if err != nil {
		panic(err)
	}
	tmpFilename := tmpFile.Name()
	if err = tmpFile.Close(); err != nil {
		panic(err)
	}

	args := []string{"build", "-o", tmpFilename}
	if exp := os.Getenv(TestExperimentFlag); exp != "" && exp != "false" {
		args = append(args, "-ldflags", "-X 'main.experimentsAllowed=yes'")
	}
	args = append(args, pkgPath)
	cmd := exec.Command("go", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err = cmd.Run()
	if err != nil {
		// The go compiler will have already produced some error messages
		// on stderr by the time we get here.
		panic(fmt.Sprintf("failed to build executable: %s", err))
	}

	return tmpFilename
}

// WorkDir() returns the binary workdir
func (b *binary) WorkDir() string {
	return b.workDir
}
