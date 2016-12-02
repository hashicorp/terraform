package command

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-getter"
	"github.com/hashicorp/terraform/config/module"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/terraform"
)

// This is the directory where our test fixtures are.
var fixtureDir = "./test-fixtures"

func init() {
	test = true

	// Expand the fixture dir on init because we change the working
	// directory in some tests.
	var err error
	fixtureDir, err = filepath.Abs(fixtureDir)
	if err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Verbose() {
		// if we're verbose, use the logging requested by TF_LOG
		logging.SetOutput()
	} else {
		// otherwise silence all logs
		log.SetOutput(ioutil.Discard)
	}

	os.Exit(m.Run())
}

func tempDir(t *testing.T) string {
	dir, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("err: %s", err)
	}

	return dir
}

func testFixturePath(name string) string {
	return filepath.Join(fixtureDir, name)
}

func testCtxConfig(p terraform.ResourceProvider) *terraform.ContextOpts {
	return &terraform.ContextOpts{
		Providers: map[string]terraform.ResourceProviderFactory{
			"test": func() (terraform.ResourceProvider, error) {
				return p, nil
			},
		},
	}
}

func testCtxConfigWithShell(p terraform.ResourceProvider, pr terraform.ResourceProvisioner) *terraform.ContextOpts {
	return &terraform.ContextOpts{
		Providers: map[string]terraform.ResourceProviderFactory{
			"test": func() (terraform.ResourceProvider, error) {
				return p, nil
			},
		},
		Provisioners: map[string]terraform.ResourceProvisionerFactory{
			"shell": func() (terraform.ResourceProvisioner, error) {
				return pr, nil
			},
		},
	}
}

func testModule(t *testing.T, name string) *module.Tree {
	mod, err := module.NewTreeModule("", filepath.Join(fixtureDir, name))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	s := &getter.FolderStorage{StorageDir: tempDir(t)}
	if err := mod.Load(s, module.GetModeGet); err != nil {
		t.Fatalf("err: %s", err)
	}

	return mod
}

// testPlan returns a non-nil noop plan.
func testPlan(t *testing.T) *terraform.Plan {
	state := terraform.NewState()
	state.RootModule().Outputs["foo"] = &terraform.OutputState{
		Type:  "string",
		Value: "foo",
	}

	return &terraform.Plan{
		Module: testModule(t, "apply"),
		State:  state,
	}
}

func testPlanFile(t *testing.T, plan *terraform.Plan) string {
	path := testTempFile(t)

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	if err := terraform.WritePlan(plan, f); err != nil {
		t.Fatalf("err: %s", err)
	}

	return path
}

func testReadPlan(t *testing.T, path string) *terraform.Plan {
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	p, err := terraform.ReadPlan(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return p
}

// testState returns a test State structure that we use for a lot of tests.
func testState() *terraform.State {
	state := &terraform.State{
		Version: 2,
		Modules: []*terraform.ModuleState{
			&terraform.ModuleState{
				Path: []string{"root"},
				Resources: map[string]*terraform.ResourceState{
					"test_instance.foo": &terraform.ResourceState{
						Type: "test_instance",
						Primary: &terraform.InstanceState{
							ID: "bar",
						},
					},
				},
				Outputs: map[string]*terraform.OutputState{},
			},
		},
	}
	state.Init()
	return state
}

func testStateFile(t *testing.T, s *terraform.State) string {
	path := testTempFile(t)

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	if err := terraform.WriteState(s, f); err != nil {
		t.Fatalf("err: %s", err)
	}

	return path
}

// testStateFileDefault writes the state out to the default statefile
// in the cwd. Use `testCwd` to change into a temp cwd.
func testStateFileDefault(t *testing.T, s *terraform.State) string {
	f, err := os.Create(DefaultStateFilename)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	if err := terraform.WriteState(s, f); err != nil {
		t.Fatalf("err: %s", err)
	}

	return DefaultStateFilename
}

// testStateFileRemote writes the state out to the remote statefile
// in the cwd. Use `testCwd` to change into a temp cwd.
func testStateFileRemote(t *testing.T, s *terraform.State) string {
	path := filepath.Join(DefaultDataDir, DefaultStateFilename)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("err: %s", err)
	}

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	if err := terraform.WriteState(s, f); err != nil {
		t.Fatalf("err: %s", err)
	}

	return path
}

// testStateOutput tests that the state at the given path contains
// the expected state string.
func testStateOutput(t *testing.T, path string, expected string) {
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	newState, err := terraform.ReadState(f)
	f.Close()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(newState.String())
	expected = strings.TrimSpace(expected)
	if actual != expected {
		t.Fatalf("expected:\n%s\nactual:\n%s", expected, actual)
	}
}

func testProvider() *terraform.MockResourceProvider {
	p := new(terraform.MockResourceProvider)
	p.DiffReturn = &terraform.InstanceDiff{}
	p.RefreshFn = func(
		info *terraform.InstanceInfo,
		s *terraform.InstanceState) (*terraform.InstanceState, error) {
		return s, nil
	}
	p.ResourcesReturn = []terraform.ResourceType{
		terraform.ResourceType{
			Name: "test_instance",
		},
	}

	return p
}

func testTempFile(t *testing.T) string {
	return filepath.Join(testTempDir(t), "state.tfstate")
}

func testTempDir(t *testing.T) string {
	d, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return d
}

// testRename renames the path to new and returns a function to defer to
// revert the rename.
func testRename(t *testing.T, base, path, new string) func() {
	if base != "" {
		path = filepath.Join(base, path)
		new = filepath.Join(base, new)
	}

	if err := os.Rename(path, new); err != nil {
		t.Fatalf("err: %s", err)
	}

	return func() {
		// Just re-rename and ignore the return value
		testRename(t, "", new, path)
	}
}

// testChdir changes the directory and returns a function to defer to
// revert the old cwd.
func testChdir(t *testing.T, new string) func() {
	old, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	if err := os.Chdir(new); err != nil {
		t.Fatalf("err: %v", err)
	}

	return func() {
		// Re-run the function ignoring the defer result
		testChdir(t, old)
	}
}

// testCwd is used to change the current working directory
// into a test directory that should be remoted after
func testCwd(t *testing.T) (string, string) {
	tmp, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("err: %v", err)
	}

	return tmp, cwd
}

// testFixCwd is used to as a defer to testDir
func testFixCwd(t *testing.T, tmp, cwd string) {
	if err := os.Chdir(cwd); err != nil {
		t.Fatalf("err: %v", err)
	}

	if err := os.RemoveAll(tmp); err != nil {
		t.Fatalf("err: %v", err)
	}
}

// testStdinPipe changes os.Stdin to be a pipe that sends the data from
// the reader before closing the pipe.
//
// The returned function should be deferred to properly clean up and restore
// the original stdin.
func testStdinPipe(t *testing.T, src io.Reader) func() {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Modify stdin to point to our new pipe
	old := os.Stdin
	os.Stdin = r

	// Copy the data from the reader to the pipe
	go func() {
		defer w.Close()
		io.Copy(w, src)
	}()

	return func() {
		// Close our read end
		r.Close()

		// Reset stdin
		os.Stdin = old
	}
}

// Modify os.Stdout to write to the given buffer. Note that this is generally
// not useful since the commands are configured to write to a cli.Ui, not
// Stdout directly. Commands like `console` though use the raw stdout.
func testStdoutCapture(t *testing.T, dst io.Writer) func() {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// Modify stdout
	old := os.Stdout
	os.Stdout = w

	// Copy
	doneCh := make(chan struct{})
	go func() {
		defer close(doneCh)
		defer r.Close()
		io.Copy(dst, r)
	}()

	return func() {
		// Close the writer end of the pipe
		w.Sync()
		w.Close()

		// Reset stdout
		os.Stdout = old

		// Wait for the data copy to complete to avoid a race reading data
		<-doneCh
	}
}
