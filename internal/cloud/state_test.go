package cloud

import (
	"bytes"
	"github.com/hashicorp/terraform/internal/states/statefile"
	"io/ioutil"
	"testing"

	"github.com/hashicorp/go-tfe"

	"github.com/hashicorp/terraform/internal/states/statemgr"
)

func TestState_impl(t *testing.T) {
	var _ statemgr.Reader = new(State)
	var _ statemgr.Writer = new(State)
	var _ statemgr.Persister = new(State)
	var _ statemgr.Refresher = new(State)
	var _ statemgr.OutputReader = new(State)
	var _ statemgr.Locker = new(State)
}

type ExpectedOutput struct {
	Name      string
	Sensitive bool
	IsNull    bool
}

func TestState_GetRootOutputValues(t *testing.T) {
	b, bCleanup := testBackendWithOutputs(t)
	defer bCleanup()

	state := &State{tfeClient: b.client, organization: b.organization, workspace: &tfe.Workspace{
		ID: "ws-abcd",
	}}
	outputs, err := state.GetRootOutputValues()

	if err != nil {
		t.Fatalf("error returned from GetRootOutputValues: %s", err)
	}

	cases := []ExpectedOutput{
		{
			Name:      "sensitive_output",
			Sensitive: true,
			IsNull:    false,
		},
		{
			Name:      "nonsensitive_output",
			Sensitive: false,
			IsNull:    false,
		},
		{
			Name:      "object_output",
			Sensitive: false,
			IsNull:    false,
		},
		{
			Name:      "list_output",
			Sensitive: false,
			IsNull:    false,
		},
	}

	if len(outputs) != len(cases) {
		t.Errorf("Expected %d item but %d were returned", len(cases), len(outputs))
	}

	for _, testCase := range cases {
		so, ok := outputs[testCase.Name]
		if !ok {
			t.Fatalf("Expected key %s but it was not found", testCase.Name)
		}
		if so.Value.IsNull() != testCase.IsNull {
			t.Errorf("Key %s does not match null expectation %v", testCase.Name, testCase.IsNull)
		}
		if so.Sensitive != testCase.Sensitive {
			t.Errorf("Key %s does not match sensitive expectation %v", testCase.Name, testCase.Sensitive)
		}
	}
}

func TestState(t *testing.T) {
	var buf bytes.Buffer
	s := statemgr.TestFullInitialState()
	sf := statefile.New(s, "stub-lineage", 2)
	err := statefile.Write(sf, &buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	data := buf.Bytes()

	state := testCloudState(t)

	jsonState, err := ioutil.ReadFile("../command/testdata/show-json-state/sensitive-variables/output.json")
	if err != nil {
		t.Fatal(err)
	}

	jsonStateOutputs := []byte(`
{
	"version": 4,
	"terraform_version": "1.3.0",
	"serial": 1,
	"lineage": "backend-change",
	"outputs": {
			"foo": {
					"type": "string",
					"value": "bar"
			}
	}
}`)

	if err := state.uploadState(state.lineage, state.serial, state.forcePush, data, jsonState, jsonStateOutputs); err != nil {
		t.Fatalf("put: %s", err)
	}

	payload, err := state.getStatePayload()
	if err != nil {
		t.Fatalf("get: %s", err)
	}
	if !bytes.Equal(payload.Data, data) {
		t.Fatalf("expected full state %q\n\ngot: %q", string(payload.Data), string(data))
	}

	if err := state.Delete(); err != nil {
		t.Fatalf("delete: %s", err)
	}

	p, err := state.getStatePayload()
	if err != nil {
		t.Fatalf("get: %s", err)
	}
	if p != nil {
		t.Fatalf("expected empty state, got: %q", string(p.Data))
	}
}
