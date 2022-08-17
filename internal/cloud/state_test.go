package cloud

import (
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

	client := &remoteClient{
		client: b.client,
		workspace: &tfe.Workspace{
			ID: "ws-abcd",
		},
	}

	state := NewState(client)
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
