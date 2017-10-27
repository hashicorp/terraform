package command

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

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
	t.Helper()

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

func metaOverridesForProvider(p terraform.ResourceProvider) *testingOverrides {
	return &testingOverrides{
		ProviderResolver: terraform.ResourceProviderResolverFixed(
			map[string]terraform.ResourceProviderFactory{
				"test": func() (terraform.ResourceProvider, error) {
					return p, nil
				},
			},
		),
	}
}

func metaOverridesForProviderAndProvisioner(p terraform.ResourceProvider, pr terraform.ResourceProvisioner) *testingOverrides {
	return &testingOverrides{
		ProviderResolver: terraform.ResourceProviderResolverFixed(
			map[string]terraform.ResourceProviderFactory{
				"test": func() (terraform.ResourceProvider, error) {
					return p, nil
				},
			},
		),
		Provisioners: map[string]terraform.ResourceProvisionerFactory{
			"shell": func() (terraform.ResourceProvisioner, error) {
				return pr, nil
			},
		},
	}
}

func testModule(t *testing.T, name string) *module.Tree {
	t.Helper()

	mod, err := module.NewTreeModule("", filepath.Join(fixtureDir, name))
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	s := &module.Storage{
		StorageDir: tempDir(t),
		Mode:       module.GetModeGet,
	}
	if err := mod.Load(s); err != nil {
		t.Fatalf("err: %s", err)
	}

	return mod
}

// testPlan returns a non-nil noop plan.
func testPlan(t *testing.T) *terraform.Plan {
	t.Helper()

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
	t.Helper()

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
	t.Helper()

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
	t.Helper()

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
	t.Helper()

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
	t.Helper()

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

// testStateRead reads the state from a file
func testStateRead(t *testing.T, path string) *terraform.State {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	newState, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return newState
}

// testStateOutput tests that the state at the given path contains
// the expected state string.
func testStateOutput(t *testing.T, path string, expected string) {
	t.Helper()

	newState := testStateRead(t, path)
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
	t.Helper()

	return filepath.Join(testTempDir(t), "state.tfstate")
}

func testTempDir(t *testing.T) string {
	t.Helper()

	d, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return d
}

// testRename renames the path to new and returns a function to defer to
// revert the rename.
func testRename(t *testing.T, base, path, new string) func() {
	t.Helper()

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
	t.Helper()

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
	t.Helper()

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
	t.Helper()

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
	t.Helper()

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
	t.Helper()

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

// testInteractiveInput configures tests so that the answers given are sent
// in order to interactive prompts. The returned function must be called
// in a defer to clean up.
func testInteractiveInput(t *testing.T, answers []string) func() {
	t.Helper()

	// Disable test mode so input is called
	test = false

	// Setup reader/writers
	testInputResponse = answers
	defaultInputReader = bytes.NewBufferString("")
	defaultInputWriter = new(bytes.Buffer)

	// Return the cleanup
	return func() {
		test = true
		testInputResponse = nil
	}
}

// testInputMap configures tests so that the given answers are returned
// for calls to Input when the right question is asked. The key is the
// question "Id" that is used.
func testInputMap(t *testing.T, answers map[string]string) func() {
	t.Helper()

	// Disable test mode so input is called
	test = false

	// Setup reader/writers
	defaultInputReader = bytes.NewBufferString("")
	defaultInputWriter = new(bytes.Buffer)

	// Setup answers
	testInputResponse = nil
	testInputResponseMap = answers

	// Return the cleanup
	return func() {
		test = true
		testInputResponseMap = nil
	}
}

// testBackendState is used to make a test HTTP server to test a configured
// backend. This returns the complete state that can be saved. Use
// `testStateFileRemote` to write the returned state.
func testBackendState(t *testing.T, s *terraform.State, c int) (*terraform.State, *httptest.Server) {
	t.Helper()

	var b64md5 string
	buf := bytes.NewBuffer(nil)

	cb := func(resp http.ResponseWriter, req *http.Request) {
		if req.Method == "PUT" {
			resp.WriteHeader(c)
			return
		}
		if s == nil {
			resp.WriteHeader(404)
			return
		}

		resp.Header().Set("Content-MD5", b64md5)
		resp.Write(buf.Bytes())
	}

	// If a state was given, make sure we calculate the proper b64md5
	if s != nil {
		enc := json.NewEncoder(buf)
		if err := enc.Encode(s); err != nil {
			t.Fatalf("err: %v", err)
		}
		md5 := md5.Sum(buf.Bytes())
		b64md5 = base64.StdEncoding.EncodeToString(md5[:16])
	}

	srv := httptest.NewServer(http.HandlerFunc(cb))

	state := terraform.NewState()
	state.Backend = &terraform.BackendState{
		Type:   "http",
		Config: map[string]interface{}{"address": srv.URL},
		Hash:   2529831861221416334,
	}

	return state, srv
}

// testRemoteState is used to make a test HTTP server to return a given
// state file that can be used for testing legacy remote state.
func testRemoteState(t *testing.T, s *terraform.State, c int) (*terraform.RemoteState, *httptest.Server) {
	t.Helper()

	var b64md5 string
	buf := bytes.NewBuffer(nil)

	cb := func(resp http.ResponseWriter, req *http.Request) {
		if req.Method == "PUT" {
			resp.WriteHeader(c)
			return
		}
		if s == nil {
			resp.WriteHeader(404)
			return
		}

		resp.Header().Set("Content-MD5", b64md5)
		resp.Write(buf.Bytes())
	}

	srv := httptest.NewServer(http.HandlerFunc(cb))
	remote := &terraform.RemoteState{
		Type:   "http",
		Config: map[string]string{"address": srv.URL},
	}

	if s != nil {
		// Set the remote data
		s.Remote = remote

		enc := json.NewEncoder(buf)
		if err := enc.Encode(s); err != nil {
			t.Fatalf("err: %v", err)
		}
		md5 := md5.Sum(buf.Bytes())
		b64md5 = base64.StdEncoding.EncodeToString(md5[:16])
	}

	return remote, srv
}

// testlockState calls a separate process to the lock the state file at path.
// deferFunc should be called in the caller to properly unlock the file.
// Since many tests change the working durectory, the sourcedir argument must be
// supplied to locate the statelocker.go source.
func testLockState(sourceDir, path string) (func(), error) {
	// build and run the binary ourselves so we can quickly terminate it for cleanup
	buildDir, err := ioutil.TempDir("", "locker")
	if err != nil {
		return nil, err
	}
	cleanFunc := func() {
		os.RemoveAll(buildDir)
	}

	source := filepath.Join(sourceDir, "statelocker.go")
	lockBin := filepath.Join(buildDir, "statelocker")

	out, err := exec.Command("go", "build", "-o", lockBin, source).CombinedOutput()
	if err != nil {
		cleanFunc()
		return nil, fmt.Errorf("%s %s", err, out)
	}

	locker := exec.Command(lockBin, path)
	pr, pw, err := os.Pipe()
	if err != nil {
		cleanFunc()
		return nil, err
	}
	defer pr.Close()
	defer pw.Close()
	locker.Stderr = pw
	locker.Stdout = pw

	if err := locker.Start(); err != nil {
		return nil, err
	}
	deferFunc := func() {
		cleanFunc()
		locker.Process.Signal(syscall.SIGTERM)
		locker.Wait()
	}

	// wait for the process to lock
	buf := make([]byte, 1024)
	n, err := pr.Read(buf)
	if err != nil {
		return deferFunc, fmt.Errorf("read from statelocker returned: %s", err)
	}

	output := string(buf[:n])
	if !strings.HasPrefix(output, "LOCKID") {
		return deferFunc, fmt.Errorf("statelocker wrote: %s", string(buf[:n]))
	}
	return deferFunc, nil
}
