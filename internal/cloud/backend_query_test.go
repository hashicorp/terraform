// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/hashicorp/cli"
	tfe "github.com/hashicorp/go-tfe"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/backend/backendrun"
	"github.com/hashicorp/terraform/internal/command/arguments"
	"github.com/hashicorp/terraform/internal/command/clistate"
	"github.com/hashicorp/terraform/internal/command/jsonformat"
	"github.com/hashicorp/terraform/internal/command/views"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"
	tftesting "github.com/hashicorp/terraform/internal/terraform/testing"
	"github.com/zclconf/go-cty/cty"
)

func testOperationQuery(t *testing.T, configDir string) (*backendrun.Operation, func(), func(*testing.T) *terminal.TestOutput) {
	t.Helper()

	return testOperationQueryWithTimeout(t, configDir, 0)
}

func testOperationQueryWithTimeout(t *testing.T, configDir string, timeout time.Duration) (*backendrun.Operation, func(), func(*testing.T) *terminal.TestOutput) {
	t.Helper()

	_, configLoader, configCleanup := tftesting.MustLoadConfigForTests(t, configDir, "tests")

	streams, done := terminal.StreamsForTesting(t)
	view := views.NewView(streams)
	stateLockerView := views.NewStateLocker(arguments.ViewHuman, view)
	operationView := views.NewQuery(arguments.ViewHuman, view).Operation()

	// Many of our tests use an overridden "null" provider that's just in-memory
	// inside the test process, not a separate plugin on disk.
	depLocks := depsfile.NewLocks()
	depLocks.SetProviderOverridden(addrs.MustParseProviderSourceString("registry.terraform.io/hashicorp/null"))

	return &backendrun.Operation{
		ConfigDir:       configDir,
		ConfigLoader:    configLoader,
		StateLocker:     clistate.NewLocker(timeout, stateLockerView),
		Type:            backendrun.OperationTypePlan,
		View:            operationView,
		DependencyLocks: depLocks,
		Query:           true,
	}, configCleanup, done
}

func TestCloud_queryBasic(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	op, configCleanup, done := testOperationQuery(t, "./testdata/query")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backendrun.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}

	output := b.CLI.(*cli.MockUi).OutputWriter.String()
	if !strings.Contains(output, "Running query in HCP Terraform") {
		t.Fatalf("expected HCP Terraform header in output: %s", output)
	}
	if !strings.Contains(output, "list.concept_pet.pets   id=") {
		t.Fatalf("expected query results in output: %s", output)
	}

	stateMgr, _ := b.StateMgr(testBackendSingleWorkspaceName)
	// An error suggests that the state was not unlocked after the operation finished
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state after successful plan: %s", err.Error())
	}
}

func TestCloud_queryJSONBasic(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	stream, close := terminal.StreamsForTesting(t)

	b.renderer = &jsonformat.Renderer{
		Streams:  stream,
		Colorize: mockColorize(),
	}

	op, configCleanup, done := testOperationQuery(t, "./testdata/query-json-basic")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	mockSROWorkspace(t, b, op.Workspace)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backendrun.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}

	outp := close(t)
	gotOut := outp.Stdout()

	expectedOut := `list.concept_pet.pets   id=large-roughy,legs=2      This is a large-roughy
list.concept_pet.pets   id=able-werewolf,legs=5     This is a able-werewolf
list.concept_pet.pets   id=complete-gannet,legs=6   This is a complete-gannet
list.concept_pet.pets   id=charming-beagle,legs=3   This is a charming-beagle
list.concept_pet.pets   id=legal-lamprey,legs=2     This is a legal-lamprey

`
	if diff := cmp.Diff(expectedOut, gotOut); diff != "" {
		t.Fatalf("expected query results output to be %s, got %s: diff: %s", expectedOut, gotOut, diff)
	}

	stateMgr, _ := b.StateMgr(testBackendSingleWorkspaceName)
	// An error suggests that the state was not unlocked after the operation finished
	if _, err := stateMgr.Lock(statemgr.NewLockInfo()); err != nil {
		t.Fatalf("unexpected error locking state after successful plan: %s", err.Error())
	}
}

func TestCloud_queryJSONWithDiags(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	stream, close := terminal.StreamsForTesting(t)

	b.renderer = &jsonformat.Renderer{
		Streams:  stream,
		Colorize: mockColorize(),
	}

	op, configCleanup, done := testOperationQuery(t, "./testdata/query-json-diag")
	defer configCleanup()
	defer done(t)

	op.Workspace = testBackendSingleWorkspaceName

	mockSROWorkspace(t, b, op.Workspace)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}

	<-run.Done()
	if run.Result != backendrun.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}

	testOut := close(t)
	output := testOut.Stdout()

	// Warning diagnostic message
	testString := "Warning: Something went wrong"
	if !strings.Contains(output, testString) {
		t.Fatalf("Expected %q to contain %q but it did not", output, testString)
	}
}

func TestCloud_queryGenerateConfigOut(t *testing.T) {
	tests := []struct {
		name          string
		setConfigOut  bool
		wantGenConfig bool
	}{
		{
			name:          "GenerateConfigOut set",
			setConfigOut:  true,
			wantGenConfig: true,
		},
		{
			name:          "GenerateConfigOut not set",
			setConfigOut:  false,
			wantGenConfig: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b, mc, bCleanup := testBackendAndMocksWithName(t)
			defer bCleanup()

			op, configCleanup, done := testOperationQuery(t, "./testdata/query")
			defer configCleanup()
			defer done(t)

			if tc.setConfigOut {
				// ValidateTargetFile errors if the file already exists, so
				// use a path inside a temp dir that does not yet exist.
				op.GenerateConfigOut = filepath.Join(t.TempDir(), "generated.tf")
			}

			op.Workspace = testBackendSingleWorkspaceName

			run, err := b.Operation(context.Background(), op)
			if err != nil {
				t.Fatalf("error starting operation: %v", err)
			}

			<-run.Done()
			if run.Result != backendrun.OperationSuccess {
				t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
			}

			got := mc.QueryRuns.CreateOptions.GenerateConfigOut
			if got == nil {
				t.Fatal("expected GenerateConfigOut to be non-nil in QueryRunCreateOptions")
			}
			if *got != tc.wantGenConfig {
				t.Errorf("GenerateConfigOut: got %v, want %v", *got, tc.wantGenConfig)
			}
		})
	}
}

// indirectHandler wraps a *func so that the mux-registered closure always
// calls whatever function is currently pointed to. This lets tests register
// the route at server-start time (before they have the MockClient) and then
// swap in the real handler body once the MockClient is available.
type indirectHandler struct {
	fn *func(http.ResponseWriter, *http.Request)
}

func (h *indirectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.fn != nil && *h.fn != nil {
		(*h.fn)(w, r)
	}
}

type operationBackendSpy struct {
	backendrun.OperationsBackend
	operation *backendrun.Operation
	calls     atomic.Int32
}

func (b *operationBackendSpy) Operation(_ context.Context, op *backendrun.Operation) (*backendrun.RunningOperation, error) {
	b.calls.Add(1)
	b.operation = op
	done, cancel := context.WithCancel(context.Background())
	cancel()
	return &backendrun.RunningOperation{
		Context: done,
		Stop:    cancel,
		Cancel:  cancel,
		Result:  backendrun.OperationSuccess,
	}, nil
}

type queryRunsCreateSpy struct {
	tfe.QueryRuns
	creates atomic.Int32
}

func (s *queryRunsCreateSpy) Create(ctx context.Context, options tfe.QueryRunCreateOptions) (*tfe.QueryRun, error) {
	s.creates.Add(1)
	return s.QueryRuns.Create(ctx, options)
}

func renderQueryRunLogsForTest(t *testing.T, logContent string) string {
	t.Helper()
	b, mc, cleanup := testBackendAndMocksWithName(t)
	defer cleanup()

	logPath := filepath.Join(t.TempDir(), "query.log")
	if err := os.WriteFile(logPath, []byte(logContent), 0o600); err != nil {
		t.Fatal(err)
	}

	stream, closeStream := terminal.StreamsForTesting(t)
	b.renderer = &jsonformat.Renderer{Streams: stream, Colorize: mockColorize()}
	run := &tfe.QueryRun{
		ID:         "qry-render-test",
		LogReadURL: "https://app.terraform.io/_archivist/qry-render-test",
		Status:     tfe.QueryRunFinished,
	}
	mc.QueryRuns.Lock()
	mc.QueryRuns.Runs[run.ID] = run
	mc.QueryRuns.logs[run.LogReadURL] = logPath
	mc.QueryRuns.Unlock()

	if err := b.renderQueryRunLogs(context.Background(), &backendrun.Operation{}, run); err != nil {
		t.Fatalf("renderQueryRunLogs returned an error: %v", err)
	}
	return closeStream(t).Stdout()
}

const policySummaryPassLog = `{"@level":"info","@message":"Policy results","type":"policy_query_summary","list_block_address":"list.test.a","overall_result":"pass","results":[{"identity":{"id":"a"},"target_address":"test.a","result":"pass","policies":[{"policy_metadata":{"policy_name":"policy.a","enforcement_level":"mandatory"},"diagnostics":[],"result":"pass"}]}],"passed_policies":[{"policy_name":"policy.a","enforcement_level":"mandatory"}]}`

// policySummaryAllNALog is a wire record where every identity produced n/a
// (i.e. no policies matched any resource in the list block).
const policySummaryAllNALog = `{"@level":"info","@message":"Policy results","type":"policy_query_summary","list_block_address":"list.test.b","overall_result":"n/a","results":[{"identity":{"id":"x"},"target_address":"test.b_0","result":"n/a","policies":[]},{"identity":{"id":"y"},"target_address":"test.b_1","result":"n/a","policies":[]}],"passed_policies":[]}`

// policySummaryMixedNALog is a wire record where one identity passed and one
// produced n/a.
const policySummaryMixedNALog = `{"@level":"info","@message":"Policy results","type":"policy_query_summary","list_block_address":"list.test.c","overall_result":"pass","results":[{"identity":{"id":"p"},"target_address":"test.c_0","result":"pass","policies":[{"policy_metadata":{"policy_name":"policy.p","enforcement_level":"mandatory"},"diagnostics":[],"result":"pass"}]},{"identity":{"id":"n"},"target_address":"test.c_1","result":"n/a","policies":[]}],"passed_policies":[{"policy_name":"policy.p","enforcement_level":"mandatory"}]}`

func TestCloud_queryWithPolicyPaths(t *testing.T) {
	var innerFn func(http.ResponseWriter, *http.Request)
	ih := &indirectHandler{fn: &innerFn}
	b, mc, cleanup := testBackendAndMocksWithHandlers(t, map[string]func(http.ResponseWriter, *http.Request){
		"/api/v2/queries": ih.ServeHTTP,
	})
	defer cleanup()

	requests := make(chan queryV2Request, 1)
	innerFn = mockQueryV2Handler(t, mc, "./testdata/query-json-policy/query.log", requests)
	v1Creates := &queryRunsCreateSpy{QueryRuns: b.client.QueryRuns}
	b.client.QueryRuns = v1Creates

	stream, closeStream := terminal.StreamsForTesting(t)
	b.renderer = &jsonformat.Renderer{Streams: stream, Colorize: mockColorize()}
	op, configCleanup, done := testOperationQuery(t, "./testdata/query-json-policy")
	defer configCleanup()
	defer done(t)
	op.Workspace = testBackendSingleWorkspaceName
	op.PolicyPaths = []string{"./policies/allow.tfpolicy.hcl", "./policies/tags.tfpolicy.hcl"}
	op.Variables = map[string]arguments.UnparsedVariableValue{
		"foo": testUnparsedVariableValue{source: terraform.ValueFromCLIArg, value: cty.StringVal("bar")},
		"bar": testUnparsedVariableValue{source: terraform.ValueFromCLIArg, value: cty.StringVal("baz")},
	}
	op.GenerateConfigOut = filepath.Join(t.TempDir(), "generated.tf")
	mockSROWorkspace(t, b, op.Workspace)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}
	<-run.Done()
	if run.Result != backendrun.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if got := v1Creates.creates.Load(); got != 0 {
		t.Fatalf("v1 query creates = %d, want 0", got)
	}

	var configVersionID string
	if len(mc.ConfigurationVersions.configVersions) != 1 {
		t.Fatalf("configuration versions = %d, want 1", len(mc.ConfigurationVersions.configVersions))
	}
	for id := range mc.ConfigurationVersions.configVersions {
		configVersionID = id
	}
	workspace := mc.Workspaces.workspaceNames[testBackendSingleWorkspaceName]
	wantRequest := queryV2Request{
		WorkspaceID:            workspace.ID,
		ConfigurationVersionID: configVersionID,
		GenerateConfigOut:      true,
		PolicyPaths:            []string{"./policies/allow.tfpolicy.hcl", "./policies/tags.tfpolicy.hcl"},
		Variables: []queryV2Variable{
			{Key: "foo", Value: `"bar"`},
			{Key: "bar", Value: `"baz"`},
		},
	}
	if diff := cmp.Diff(wantRequest, <-requests, cmpopts.SortSlices(func(a, b queryV2Variable) bool {
		return a.Key < b.Key
	})); diff != "" {
		t.Fatalf("unexpected v2 request (-want +got):\n%s", diff)
	}
	output := closeStream(t).Stdout()
	for _, want := range []string{"list.concept_pet.pets", "Evaluated 1 policies.", "id=large-roughy  Passed"} {
		if !strings.Contains(output, want) {
			t.Errorf("query output does not contain %q:\n%s", want, output)
		}
	}
}

func TestCloud_queryWithoutPolicyPaths(t *testing.T) {
	var v2Posts atomic.Int32
	b, bCleanup := testBackendWithHandlers(t, map[string]func(http.ResponseWriter, *http.Request){
		"/api/v2/queries": func(w http.ResponseWriter, _ *http.Request) {
			v2Posts.Add(1)
			w.WriteHeader(http.StatusInternalServerError)
		},
	})
	defer bCleanup()
	v1Creates := &queryRunsCreateSpy{QueryRuns: b.client.QueryRuns}
	b.client.QueryRuns = v1Creates

	stream, closeStream := terminal.StreamsForTesting(t)
	defer closeStream(t)
	b.renderer = &jsonformat.Renderer{Streams: stream, Colorize: mockColorize()}
	op, configCleanup, done := testOperationQuery(t, "./testdata/query-json-basic")
	defer configCleanup()
	defer done(t)
	op.Workspace = testBackendSingleWorkspaceName
	mockSROWorkspace(t, b, op.Workspace)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}
	<-run.Done()
	if run.Result != backendrun.OperationSuccess {
		t.Fatalf("operation failed: %s", b.CLI.(*cli.MockUi).ErrorWriter.String())
	}
	if got := v1Creates.creates.Load(); got != 1 {
		t.Fatalf("v1 query creates = %d, want 1", got)
	}
	if got := v2Posts.Load(); got != 0 {
		t.Fatalf("v2 query creates = %d, want 0", got)
	}
}

func TestCloud_createQueryRunV2RejectsMissingResponseData(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
	}{
		{name: "missing data", body: `{}`},
		{name: "missing id", body: `{"data":{"type":"queries"}}`},
		{name: "empty id", body: `{"data":{"type":"queries","id":""}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			handlers := map[string]func(http.ResponseWriter, *http.Request){
				"/api/v2/queries": func(w http.ResponseWriter, _ *http.Request) {
					w.Header().Set("Content-Type", "application/vnd.api+json")
					w.WriteHeader(http.StatusCreated)
					fmt.Fprint(w, tc.body)
				},
			}
			b, cleanup := testBackendWithHandlers(t, handlers)
			defer cleanup()

			generateConfigOut := false
			_, err := b.createQueryRunV2(context.Background(), tfe.QueryRunCreateOptions{
				ConfigurationVersion: &tfe.ConfigurationVersion{ID: "cv-test"},
				Workspace:            &tfe.Workspace{ID: "ws-test"},
				GenerateConfigOut:    &generateConfigOut,
			}, []string{"policy.tfpolicy.hcl"})
			if err == nil || !strings.Contains(err.Error(), "empty response from query run create") {
				t.Fatalf("got error %v, want empty response error", err)
			}
		})
	}
}

func writeQueryV2Response(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/vnd.api+json")
	w.WriteHeader(http.StatusCreated)
	fmt.Fprint(w, `{"data":{"type":"queries","id":"qry-test"}}`)
}

func assertRetriedQueryV2Request(t *testing.T, bodies []string, wantGenerateConfigOut bool) {
	t.Helper()
	if len(bodies) < 2 {
		t.Fatalf("request bodies = %d, want at least 2", len(bodies))
	}
	for _, body := range bodies[1:] {
		if body != bodies[0] {
			t.Fatalf("retried request body changed\nfirst: %s\nretry: %s", bodies[0], body)
		}
	}
	var request struct {
		Data struct {
			Attributes struct {
				GenerateConfigOut *bool `json:"generate-config-out"`
			} `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(bodies[0]), &request); err != nil {
		t.Fatalf("decoding request body: %s", err)
	}
	if got := request.Data.Attributes.GenerateConfigOut; got == nil || *got != wantGenerateConfigOut {
		t.Fatalf("generate-config-out = %v, want %t", got, wantGenerateConfigOut)
	}
}

func TestCloud_queryV2RetriesCreate(t *testing.T) {
	for _, tc := range []struct {
		status   int
		failures int32
	}{
		{status: http.StatusTooManyRequests, failures: 1},
		{status: http.StatusInternalServerError, failures: 2},
	} {
		t.Run(http.StatusText(tc.status), func(t *testing.T) {
			var attempts atomic.Int32
			var bodies []string
			b, cleanup := testBackendWithHandlers(t, map[string]func(http.ResponseWriter, *http.Request){
				"/api/v2/queries": func(w http.ResponseWriter, r *http.Request) {
					if r.Method != http.MethodPost {
						t.Errorf("request method = %s, want POST", r.Method)
					}
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Errorf("reading request body: %s", err)
					}
					bodies = append(bodies, string(body))
					if attempts.Add(1) <= tc.failures {
						w.Header().Set("Retry-After", "0.001")
						w.WriteHeader(tc.status)
						return
					}
					writeQueryV2Response(w)
				},
			})
			defer cleanup()
			generateConfigOut := false
			_, err := b.createQueryRunV2(context.Background(), tfe.QueryRunCreateOptions{
				ConfigurationVersion: &tfe.ConfigurationVersion{ID: "cv-test"},
				Workspace:            &tfe.Workspace{ID: "ws-test"},
				GenerateConfigOut:    &generateConfigOut,
			}, []string{"policy.tfpolicy.hcl"})
			if err != nil {
				t.Fatalf("createQueryRunV2 returned an error: %s", err)
			}
			if got, want := attempts.Load(), tc.failures+1; got != want {
				t.Fatalf("create attempts = %d, want %d", got, want)
			}
			assertRetriedQueryV2Request(t, bodies, false)
			if tc.status == http.StatusInternalServerError {
				output := b.CLI.(*cli.MockUi).OutputWriter.String()
				if !strings.Contains(output, "Trying to restore the connection") {
					t.Fatalf("retry hook output missing from:\n%s", output)
				}
			}
		})
	}
}

func TestCloud_queryV2RetriesTransportErrors(t *testing.T) {
	var attempts atomic.Int32
	var bodies []string
	b, cleanup := testBackendWithHandlers(t, map[string]func(http.ResponseWriter, *http.Request){
		"/api/v2/queries": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("request method = %s, want POST", r.Method)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Errorf("reading request body: %s", err)
			}
			bodies = append(bodies, string(body))
			if attempts.Add(1) <= 2 {
				conn, _, err := w.(http.Hijacker).Hijack()
				if err != nil {
					t.Errorf("failed to hijack connection: %s", err)
					return
				}
				conn.Close()
				return
			}
			writeQueryV2Response(w)
		},
	})
	defer cleanup()

	generateConfigOut := false
	_, err := b.createQueryRunV2(context.Background(), tfe.QueryRunCreateOptions{
		ConfigurationVersion: &tfe.ConfigurationVersion{ID: "cv-test"},
		Workspace:            &tfe.Workspace{ID: "ws-test"},
		GenerateConfigOut:    &generateConfigOut,
	}, []string{"policy.tfpolicy.hcl"})
	if err != nil {
		t.Fatalf("createQueryRunV2 returned an error: %s", err)
	}
	if got := attempts.Load(); got != 3 {
		t.Fatalf("create attempts = %d, want 3", got)
	}
	assertRetriedQueryV2Request(t, bodies, false)
	if output := b.CLI.(*cli.MockUi).OutputWriter.String(); !strings.Contains(output, "Trying to restore the connection") {
		t.Fatalf("retry hook output missing from:\n%s", output)
	}
}

func TestCloud_renderQueryRunLogsPolicySummaries(t *testing.T) {
	passOutput := `Evaluated 1 policies.

Policy results for list.test.a - Passed
  id=a  Passed
`
	malformed := `{"@level":"info","@message":"Policy results","type":"policy_query_summary","list_block_address":"","overall_result":"pass","results":[],"passed_policies":[]}`
	plainQuery := `{"@level":"info","@message":"Starting query","type":"list_start","list_start":{"address":"list.test.a"}}
{"@level":"info","@message":"Result found","type":"list_resource_found","list_resource_found":{"address":"list.test.a","display_name":"Alpha","identity":{"id":"a"}}}
{"@level":"info","@message":"List complete","type":"list_complete","list_complete":{"address":"list.test.a"}}`

	allNAOutput := `Evaluated 0 policies.

Policy results for list.test.b - N/A
  id=x  N/A
  id=y  N/A
`
	mixedNAOutput := `Evaluated 1 policies.

Policy results for list.test.c - Passed
  id=p  Passed
  id=n  N/A
`

	for _, tc := range []struct {
		name    string
		records string
		want    string
	}{
		{name: "query and summary", records: plainQuery + "\n" + policySummaryPassLog, want: "list.test.a   id=a   Alpha\n\n" + passOutput},
		{name: "malformed then valid", records: malformed + "\n" + policySummaryPassLog, want: passOutput},
		{name: "all n/a summary", records: policySummaryAllNALog, want: allNAOutput},
		{name: "mixed n/a summary", records: policySummaryMixedNALog, want: mixedNAOutput},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := renderQueryRunLogsForTest(t, tc.records); got != tc.want {
				t.Fatalf("unexpected rendered logs\nwant:\n%q\n\ngot:\n%q", tc.want, got)
			}
		})
	}
}

func TestCloud_queryWithNilClientV2(t *testing.T) {
	b, _, bCleanup := testBackendAndMocksWithName(t)
	defer bCleanup()
	v1Creates := &queryRunsCreateSpy{QueryRuns: b.client.QueryRuns}
	b.client.QueryRuns = v1Creates
	b.clientV2 = nil

	op, configCleanup, done := testOperationQuery(t, "./testdata/query-json-basic")
	defer configCleanup()

	op.Workspace = testBackendSingleWorkspaceName
	op.PolicyPaths = []string{"./policies/allow.tfpolicy.hcl"}
	mockSROWorkspace(t, b, op.Workspace)

	run, err := b.Operation(context.Background(), op)
	if err != nil {
		t.Fatalf("error starting operation: %v", err)
	}
	<-run.Done()

	if run.Result == backendrun.OperationSuccess {
		t.Fatal("operation succeeded without a v2 client")
	}
	if got := v1Creates.creates.Load(); got != 0 {
		t.Fatalf("v1 query creates = %d, want 0", got)
	}
	if stderr := done(t).Stderr(); !strings.Contains(stderr, "cannot forward -policies") {
		t.Errorf("expected error output to mention \"cannot forward -policies\", got: %s", stderr)
	}
}

func TestCloud_localDelegationPolicyClientRequirements(t *testing.T) {
	for _, tc := range []struct {
		name           string
		workspaceLocal bool
		forceLocal     bool
		op             *backendrun.Operation
		wantError      bool
	}{
		{
			name:           "workspace local query without client",
			workspaceLocal: true,
			wantError:      true,
			op: &backendrun.Operation{
				Type:        backendrun.OperationTypePlan,
				Query:       true,
				PolicyPaths: []string{"./policies/allow.tfpolicy.hcl"},
			},
		},
		{
			name:       "forced local query without client",
			forceLocal: true,
			wantError:  true,
			op: &backendrun.Operation{
				Type:        backendrun.OperationTypePlan,
				Query:       true,
				PolicyPaths: []string{"./policies/allow.tfpolicy.hcl"},
			},
		},
		{
			name:       "query with policies and client",
			forceLocal: true,
			op: &backendrun.Operation{
				Type:         backendrun.OperationTypePlan,
				Query:        true,
				PolicyPaths:  []string{"./policies/allow.tfpolicy.hcl"},
				PolicyClient: &policy.MockClient{},
			},
		},
		{
			name:       "query without policies",
			forceLocal: true,
			op:         &backendrun.Operation{Type: backendrun.OperationTypePlan, Query: true},
		},
		{
			name:       "plan with policy paths",
			forceLocal: true,
			op: &backendrun.Operation{
				Type:        backendrun.OperationTypePlan,
				PolicyPaths: []string{"./policies/allow.tfpolicy.hcl"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b, _, cleanup := testBackendAndMocksWithName(t)
			defer cleanup()
			b.forceLocal = tc.forceLocal
			local := &operationBackendSpy{}
			b.local = local
			tc.op.Workspace = testBackendSingleWorkspaceName

			ctx := context.Background()
			if tc.workspaceLocal {
				_, err := b.client.Workspaces.Update(ctx, b.Organization, b.WorkspaceMapping.Name, tfe.WorkspaceUpdateOptions{
					ExecutionMode: tfe.String("local"),
				})
				if err != nil {
					t.Fatalf("error updating workspace execution mode: %v", err)
				}
			}

			run, err := b.Operation(ctx, tc.op)
			if tc.wantError {
				if run != nil || err == nil || !strings.Contains(err.Error(), "policy engine is unavailable") {
					t.Fatalf("run = %#v, error = %v; want unavailable policy engine error", run, err)
				}
				if got := local.calls.Load(); got != 0 {
					t.Fatalf("local operations = %d, want 0", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Operation() returned error: %v", err)
			}
			<-run.Done()
			if run.Result != backendrun.OperationSuccess {
				t.Fatal("local operation failed")
			}
			if local.operation != tc.op {
				t.Fatal("operation was not delegated to the local backend")
			}
			if got := local.calls.Load(); got != 1 {
				t.Fatalf("local operations = %d, want 1", got)
			}
		})
	}
}
