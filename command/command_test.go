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

	"github.com/hashicorp/terraform/addrs"
	"github.com/hashicorp/terraform/configs"
	"github.com/hashicorp/terraform/configs/configload"
	"github.com/hashicorp/terraform/configs/configschema"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/plans"
	"github.com/hashicorp/terraform/plans/planfile"
	"github.com/hashicorp/terraform/providers"
	"github.com/hashicorp/terraform/provisioners"
	"github.com/hashicorp/terraform/states"
	"github.com/hashicorp/terraform/states/statefile"
	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/hashicorp/terraform/terraform"
	"github.com/hashicorp/terraform/version"
	"github.com/zclconf/go-cty/cty"

	backendInit "github.com/hashicorp/terraform/backend/init"
)

// This is the directory where our test fixtures are.
var fixtureDir = "./test-fixtures"

// a top level temp directory which will be cleaned after all tests
var testingDir string

func init() {
	test = true

	// Initialize the backends
	backendInit.Init(nil)

	// Expand the fixture dir on init because we change the working
	// directory in some tests.
	var err error
	fixtureDir, err = filepath.Abs(fixtureDir)
	if err != nil {
		panic(err)
	}

	testingDir, err = ioutil.TempDir(testingDir, "tf")
	if err != nil {
		panic(err)
	}
}

func TestMain(m *testing.M) {
	defer os.RemoveAll(testingDir)

	flag.Parse()
	if testing.Verbose() {
		// if we're verbose, use the logging requested by TF_LOG
		logging.SetOutput()
	} else {
		// otherwise silence all logs
		log.SetOutput(ioutil.Discard)
	}

	// Make sure backend init is initialized, since our tests tend to assume it.
	backendInit.Init(nil)

	os.Exit(m.Run())
}

func tempDir(t *testing.T) string {
	t.Helper()

	dir, err := ioutil.TempDir(testingDir, "tf")
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

func metaOverridesForProvider(p providers.Interface) *testingOverrides {
	return &testingOverrides{
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"test": providers.FactoryFixed(p),
			},
		),
	}
}

func metaOverridesForProviderAndProvisioner(p providers.Interface, pr provisioners.Interface) *testingOverrides {
	return &testingOverrides{
		ProviderResolver: providers.ResolverFixed(
			map[string]providers.Factory{
				"test": providers.FactoryFixed(p),
			},
		),
		Provisioners: map[string]provisioners.Factory{
			"shell": provisioners.FactoryFixed(pr),
		},
	}
}

func testModule(t *testing.T, name string) *configs.Config {
	t.Helper()
	c, _ := testModuleWithSnapshot(t, name)
	return c
}

func testModuleWithSnapshot(t *testing.T, name string) (*configs.Config, *configload.Snapshot) {
	t.Helper()

	dir := filepath.Join(fixtureDir, name)

	// FIXME: We're not dealing with the cleanup function here because
	// this testModule function is used all over and so we don't want to
	// change its interface at this late stage.
	loader, _ := configload.NewLoaderForTests(t)

	// Test modules usually do not refer to remote sources, and for local
	// sources only this ultimately just records all of the module paths
	// in a JSON file so that we can load them below.
	diags := loader.InstallModules(dir, true, configload.InstallHooksImpl{})
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	config, snap, diags := loader.LoadConfigWithSnapshot(dir)
	if diags.HasErrors() {
		t.Fatal(diags.Error())
	}

	return config, snap
}

// testPlan returns a non-nil noop plan.
func testPlan(t *testing.T) *plans.Plan {
	t.Helper()

	// This is what an empty configuration block would look like after being
	// decoded with the schema of the "local" backend.
	backendConfig := cty.ObjectVal(map[string]cty.Value{
		"path":          cty.NullVal(cty.String),
		"workspace_dir": cty.NullVal(cty.String),
	})
	backendConfigRaw, err := plans.NewDynamicValue(backendConfig, backendConfig.Type())
	if err != nil {
		t.Fatal(err)
	}

	return &plans.Plan{
		Backend: plans.Backend{
			// This is just a placeholder so that the plan file can be written
			// out. Caller may wish to override it to something more "real"
			// where the plan will actually be subsequently applied.
			Type:   "local",
			Config: backendConfigRaw,
		},
		Changes: plans.NewChanges(),
	}
}

func testPlanFile(t *testing.T, configSnap *configload.Snapshot, state *states.State, plan *plans.Plan) string {
	t.Helper()

	stateFile := &statefile.File{
		Lineage:          "",
		State:            state,
		TerraformVersion: version.SemVer,
	}

	path := testTempFile(t)
	err := planfile.Create(path, configSnap, stateFile, plan)
	if err != nil {
		t.Fatalf("failed to create temporary plan file: %s", err)
	}

	return path
}

// testPlanFileNoop is a shortcut function that creates a plan file that
// represents no changes and returns its path. This is useful when a test
// just needs any plan file, and it doesn't matter what is inside it.
func testPlanFileNoop(t *testing.T) string {
	snap := &configload.Snapshot{
		Modules: map[string]*configload.SnapshotModule{
			"": {
				Dir: ".",
				Files: map[string][]byte{
					"main.tf": nil,
				},
			},
		},
	}
	state := states.NewState()
	plan := testPlan(t)
	return testPlanFile(t, snap, state, plan)
}

func testReadPlan(t *testing.T, path string) *plans.Plan {
	t.Helper()

	f, err := planfile.Open(path)
	if err != nil {
		t.Fatalf("error opening plan file %q: %s", path, err)
	}
	defer f.Close()

	p, err := f.ReadPlan()
	if err != nil {
		t.Fatalf("error reading plan from plan file %q: %s", path, err)
	}

	return p
}

// testState returns a test State structure that we use for a lot of tests.
func testState() *states.State {
	return states.BuildState(func(s *states.SyncState) {
		s.SetResourceInstanceCurrent(
			addrs.Resource{
				Mode: addrs.ManagedResourceMode,
				Type: "test_instance",
				Name: "foo",
			}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance),
			&states.ResourceInstanceObjectSrc{
				// The weird whitespace here is reflective of how this would
				// get written out in a real state file, due to the indentation
				// of all of the containing wrapping objects and arrays.
				AttrsJSON: []byte("{\n            \"id\": \"bar\"\n          }"),
				Status:    states.ObjectReady,
			},
			addrs.ProviderConfig{
				Type: "test",
			}.Absolute(addrs.RootModuleInstance),
		)
	})
}

// writeStateForTesting is a helper that writes the given naked state to the
// given writer, generating a stub *statefile.File wrapper which is then
// immediately discarded.
func writeStateForTesting(state *states.State, w io.Writer) error {
	sf := &statefile.File{
		Serial:  0,
		Lineage: "fake-for-testing",
		State:   state,
	}
	return statefile.Write(sf, w)
}

// testStateMgrCurrentLineage returns the current lineage for the given state
// manager, or the empty string if it does not use lineage. This is primarily
// for testing against the local backend, which always supports lineage.
func testStateMgrCurrentLineage(mgr statemgr.Persistent) string {
	if pm, ok := mgr.(statemgr.PersistentMeta); ok {
		m := pm.StateSnapshotMeta()
		return m.Lineage
	}
	return ""
}

// markStateForMatching is a helper that writes a specific marker value to
// a state so that it can be recognized later with getStateMatchingMarker.
//
// Internally this just sets a root module output value called "testing_mark"
// to the given string value. If the state is being checked in other ways,
// the test code may need to compensate for the addition or overwriting of this
// special output value name.
//
// The given mark string is returned verbatim, to allow the following pattern
// in tests:
//
//     mark := markStateForMatching(state, "foo")
//     // (do stuff to the state)
//     assertStateHasMarker(state, mark)
func markStateForMatching(state *states.State, mark string) string {
	state.RootModule().SetOutputValue("testing_mark", cty.StringVal(mark), false)
	return mark
}

// getStateMatchingMarker is used with markStateForMatching to retrieve the
// mark string previously added to the given state. If no such mark is present,
// the result is an empty string.
func getStateMatchingMarker(state *states.State) string {
	os := state.RootModule().OutputValues["testing_mark"]
	if os == nil {
		return ""
	}
	v := os.Value
	if v.Type() == cty.String && v.IsKnown() && !v.IsNull() {
		return v.AsString()
	}
	return ""
}

// stateHasMarker is a helper around getStateMatchingMarker that also includes
// the equality test, for more convenient use in test assertion branches.
func stateHasMarker(state *states.State, want string) bool {
	return getStateMatchingMarker(state) == want
}

// assertStateHasMarker wraps stateHasMarker to automatically generate a
// fatal test result (i.e. t.Fatal) if the marker doesn't match.
func assertStateHasMarker(t *testing.T, state *states.State, want string) {
	if !stateHasMarker(state, want) {
		t.Fatalf("wrong state marker\ngot:  %q\nwant: %q", getStateMatchingMarker(state), want)
	}
}

func testStateFile(t *testing.T, s *states.State) string {
	t.Helper()

	path := testTempFile(t)

	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("failed to create temporary state file %s: %s", path, err)
	}
	defer f.Close()

	err = writeStateForTesting(s, f)
	if err != nil {
		t.Fatalf("failed to write state to temporary file %s: %s", path, err)
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
func testStateRead(t *testing.T, path string) *states.State {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	sf, err := statefile.Read(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return sf.State
}

// testDataStateRead reads a "data state", which is a file format resembling
// our state format v3 that is used only to track current backend settings.
//
// This old format still uses *terraform.State, but should be replaced with
// a more specialized type in a later release.
func testDataStateRead(t *testing.T, path string) *terraform.State {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer f.Close()

	s, err := terraform.ReadState(f)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	return s
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

func testProvider() *terraform.MockProvider {
	p := new(terraform.MockProvider)
	p.PlanResourceChangeResponse = providers.PlanResourceChangeResponse{
		PlannedState: cty.EmptyObjectVal,
	}
	p.ReadResourceFn = func(req providers.ReadResourceRequest) providers.ReadResourceResponse {
		return providers.ReadResourceResponse{
			NewState: req.PriorState,
		}
	}
	return p
}

func testTempFile(t *testing.T) string {
	t.Helper()

	return filepath.Join(testTempDir(t), "state.tfstate")
}

func testTempDir(t *testing.T) string {
	t.Helper()

	d, err := ioutil.TempDir(testingDir, "tf")
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

	tmp, err := ioutil.TempDir(testingDir, "tf")
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
//
// When using this function, the configuration fixture for the test must
// include an empty configuration block for the HTTP backend, like this:
//
// terraform {
//   backend "http" {
//   }
// }
//
// If such a block isn't present, or if it isn't empty, then an error will
// be returned about the backend configuration having changed and that
// "terraform init" must be run, since the test backend config cache created
// by this function contains the hash for an empty configuration.
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

	backendConfig := &configs.Backend{
		Type:   "http",
		Config: configs.SynthBody("<testBackendState>", map[string]cty.Value{}),
	}
	b := backendinit.Backend("http")()
	configSchema := b.ConfigSchema()
	hash := backendConfig.Hash(configSchema)

	state := terraform.NewState()
	state.Backend = &terraform.BackendState{
		Type:      "http",
		ConfigRaw: json.RawMessage(fmt.Sprintf(`{"address":%q}`, srv.URL)),
		Hash:      hash,
	}

	return state, srv
}

// testRemoteState is used to make a test HTTP server to return a given
// state file that can be used for testing legacy remote state.
//
// The return values are a *terraform.State instance that should be written
// as the "data state" (really: backend state) and the server that the
// returned data state refers to.
func testRemoteState(t *testing.T, s *states.State, c int) (*terraform.State, *httptest.Server) {
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

	retState := terraform.NewState()

	srv := httptest.NewServer(http.HandlerFunc(cb))
	b := &terraform.BackendState{
		Type: "http",
	}
	b.SetConfig(cty.ObjectVal(map[string]cty.Value{
		"address": cty.StringVal(srv.URL),
	}), &configschema.Block{
		Attributes: map[string]*configschema.Attribute{
			"address": {
				Type:     cty.String,
				Required: true,
			},
		},
	})
	retState.Backend = b

	if s != nil {
		err := statefile.Write(&statefile.File{State: s}, buf)
		if err != nil {
			t.Fatalf("failed to write initial state: %v", err)
		}
	}

	return retState, srv
}

// testlockState calls a separate process to the lock the state file at path.
// deferFunc should be called in the caller to properly unlock the file.
// Since many tests change the working durectory, the sourcedir argument must be
// supplied to locate the statelocker.go source.
func testLockState(sourceDir, path string) (func(), error) {
	// build and run the binary ourselves so we can quickly terminate it for cleanup
	buildDir, err := ioutil.TempDir(testingDir, "locker")
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

// normalizeJSON removes all insignificant whitespace from the given JSON buffer
// and returns it as a string for easier comparison.
func normalizeJSON(t *testing.T, src []byte) string {
	t.Helper()
	var buf bytes.Buffer
	err := json.Compact(&buf, src)
	if err != nil {
		t.Fatalf("error normalizing JSON: %s", err)
	}
	return buf.String()
}
