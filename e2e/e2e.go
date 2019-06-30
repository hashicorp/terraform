package e2e

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
)

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
func NewBinary(binaryPath, workingDir string) *binary {
	tmpDir, err := ioutil.TempDir("", "binary-e2etest")
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
	args := make([]string, len(parts)+1)
	args[0] = b.workDir
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
	return ioutil.ReadFile(flatPath)
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
	f, err := b.OpenFile("terraform.tfstate")
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

// Close cleans up the temporary resources associated with the object,
// including its working directory. It is not valid to call Cmd or Run
// after Close returns.
//
// This method does _not_ stop any running child processes. It's the
// caller's responsibility to also terminate those _before_ closing the
// underlying binary object.
//
// This function is designed to run under "defer", so it doesn't actually
// do any error handling and will leave dangling temporary files on disk
// if any errors occur while cleaning up.
func (b *binary) Close() {
	os.RemoveAll(b.workDir)
}

func GoBuild(pkgPath, tmpPrefix string) string {
	tmpFile, err := ioutil.TempFile("", tmpPrefix)
	if err != nil {
		panic(err)
	}
	tmpFilename := tmpFile.Name()
	if err = tmpFile.Close(); err != nil {
		panic(err)
	}

	cmd := exec.Command(
		"go", "build",
		"-mod=vendor",
		"-o", tmpFilename,
		pkgPath,
	)
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
