package e2etest

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	tfcore "github.com/hashicorp/terraform/terraform"
)

var terraformBin string

func TestMain(m *testing.M) {
	teardown := setup()
	code := m.Run()
	teardown()
	os.Exit(code)
}

func setup() func() {
	if terraformBin != "" {
		// this is pre-set when we're running in a binary produced from
		// the make-archive.sh script, since that builds a ready-to-go
		// binary into the archive. However, we do need to turn it into
		// an absolute path so that we can find it when we change the
		// working directory during tests.
		var err error
		terraformBin, err = filepath.Abs(terraformBin)
		if err != nil {
			panic(fmt.Sprintf("failed to find absolute path of terraform executable: %s", err))
		}
		return func() {}
	}

	tmpFile, err := ioutil.TempFile("", "terraform")
	if err != nil {
		panic(err)
	}
	tmpFilename := tmpFile.Name()
	if err = tmpFile.Close(); err != nil {
		panic(err)
	}

	cmd := exec.Command(
		"go", "build",
		"-o", tmpFilename,
		"github.com/hashicorp/terraform",
	)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err = cmd.Run()
	if err != nil {
		// The go compiler will have already produced some error messages
		// on stderr by the time we get here.
		panic(fmt.Sprintf("failed to build terraform executable: %s", err))
	}

	// Make the executable available for use in tests
	terraformBin = tmpFilename

	return func() {
		os.Remove(tmpFilename)
	}
}

func canAccessNetwork() bool {
	// We re-use the flag normally used for acceptance tests since that's
	// established as a way to opt-in to reaching out to real systems that
	// may suffer transient errors.
	return os.Getenv("TF_ACC") != ""
}

func skipIfCannotAccessNetwork(t *testing.T) {
	if !canAccessNetwork() {
		t.Skip("network access not allowed; use TF_ACC=1 to enable")
	}
}

// Type terraform represents the combination of a compiled Terraform binary
// and a temporary working directory to run it in.
//
// This is the main harness for tests in this package.
type terraform struct {
	bin string
	dir string
}

// newTerraform prepares a temporary directory containing the files from the
// given fixture and returns an instance of type terraform that can run
// the generated Terraform binary in that directory.
//
// If the temporary directory cannot be created, a fixture of the given name
// cannot be found, or if an error occurs while _copying_ the fixture files,
// this function will panic. Tests should be written to assume that this
// function always succeeds.
func newTerraform(fixtureName string) *terraform {
	tmpDir, err := ioutil.TempDir("", "terraform-e2etest")
	if err != nil {
		panic(err)
	}

	// For our purposes here we do a very simplistic file copy that doesn't
	// attempt to preserve file permissions, attributes, alternate data
	// streams, etc. Since we only have to deal with our own fixtures in
	// the test-fixtures subdir, we know we don't need to deal with anything
	// of this nature.
	srcDir := filepath.Join("test-fixtures", fixtureName)
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == srcDir {
			// nothing to do at the root
			return nil
		}

		srcFn := path

		path, err = filepath.Rel(srcDir, path)
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

	return &terraform{
		bin: terraformBin,
		dir: tmpDir,
	}
}

// Cmd returns an exec.Cmd pre-configured to run the generated Terraform
// binary with the given arguments in the temporary working directory.
//
// The returned object can be mutated by the caller to customize how the
// process will be run, before calling Run.
func (t *terraform) Cmd(args ...string) *exec.Cmd {
	cmd := exec.Command(t.bin, args...)
	cmd.Dir = t.dir
	cmd.Env = os.Environ()

	// Disable checkpoint since we don't want to harass that service when
	// our tests run. (This does, of course, mean we can't actually do
	// end-to-end testing of our Checkpoint interactions.)
	cmd.Env = append(cmd.Env, "CHECKPOINT_DISABLE=1")

	return cmd
}

// Run executes the generated Terraform binary with the given arguments
// and returns the bytes that it wrote to both stdout and stderr.
//
// This is a simple way to run Terraform for non-interactive commands
// that don't need any special environment variables. For more complex
// situations, use Cmd and customize the command before running it.
func (t *terraform) Run(args ...string) (stdout, stderr string, err error) {
	cmd := t.Cmd(args...)
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
func (t *terraform) Path(parts ...string) string {
	args := make([]string, len(parts)+1)
	args[0] = t.dir
	args = append(args, parts...)
	return filepath.Join(args...)
}

// OpenFile is a helper for easily opening a file from the working directory
// for reading.
func (t *terraform) OpenFile(path ...string) (*os.File, error) {
	flatPath := t.Path(path...)
	return os.Open(flatPath)
}

// ReadFile is a helper for easily reading a whole file from the working
// directory.
func (t *terraform) ReadFile(path ...string) ([]byte, error) {
	flatPath := t.Path(path...)
	return ioutil.ReadFile(flatPath)
}

// FileExists is a helper for easily testing whether a particular file
// exists in the working directory.
func (t *terraform) FileExists(path ...string) bool {
	flatPath := t.Path(path...)
	_, err := os.Stat(flatPath)
	return !os.IsNotExist(err)
}

// LocalState is a helper for easily reading the local backend's state file
// terraform.tfstate from the working directory.
func (t *terraform) LocalState() (*tfcore.State, error) {
	f, err := t.OpenFile("terraform.tfstate")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return tfcore.ReadState(f)
}

// Plan is a helper for easily reading a plan file from the working directory.
func (t *terraform) Plan(path ...string) (*tfcore.Plan, error) {
	f, err := t.OpenFile(path...)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return tfcore.ReadPlan(f)
}

// SetLocalState is a helper for easily writing to the file the local backend
// uses for state in the working directory. This does not go through the
// actual local backend code, so processing such as management of serials
// does not apply and the given state will simply be written verbatim.
func (t *terraform) SetLocalState(state *tfcore.State) error {
	path := t.Path("terraform.tfstate")
	f, err := os.OpenFile(path, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() {
		err := f.Close()
		if err != nil {
			panic(fmt.Sprintf("failed to close state file after writing: %s", err))
		}
	}()

	return tfcore.WriteState(state, f)
}

// Close cleans up the temporary resources associated with the object,
// including its working directory. It is not valid to call Cmd or Run
// after Close returns.
//
// This method does _not_ stop any running child processes. It's the
// caller's responsibility to also terminate those _before_ closing the
// underlying terraform object.
//
// This function is designed to run under "defer", so it doesn't actually
// do any error handling and will leave dangling temporary files on disk
// if any errors occur while cleaning up.
func (t *terraform) Close() {
	os.RemoveAll(t.dir)
}
