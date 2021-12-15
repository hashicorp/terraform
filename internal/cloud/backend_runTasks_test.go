package cloud

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/go-tfe"
)

type testIntegrationOutput struct {
	ctx    *IntegrationContext
	output *strings.Builder
	t      *testing.T
}

var _ IntegrationOutputWriter = (*testIntegrationOutput)(nil) // Compile time check

func (s *testIntegrationOutput) End() {
	s.output.WriteString("END\n")
}

func (s *testIntegrationOutput) SubOutput(str string) {
	s.output.WriteString(s.ctx.B.Colorize().Color("[reset]│ "+str) + "\n")
}

func (s *testIntegrationOutput) Output(str string) {
	s.output.WriteString(s.ctx.B.Colorize().Color("[reset]│ ") + str + "\n")
}

func (s *testIntegrationOutput) OutputElapsed(message string, maxMessage int) {
	s.output.WriteString("PENDING MESSAGE: " + message)
}

func newMockIntegrationContext(b *Cloud, t *testing.T) (*IntegrationContext, *testIntegrationOutput) {
	ctx := context.Background()

	// Retrieve the workspace used to run this operation in.
	w, err := b.client.Workspaces.Read(ctx, b.organization, b.WorkspaceMapping.Name)
	if err != nil {
		t.Fatalf("error retrieving workspace: %v", err)
	}

	// Create a new configuration version.
	c, err := b.client.ConfigurationVersions.Create(ctx, w.ID, tfe.ConfigurationVersionCreateOptions{})
	if err != nil {
		t.Fatalf("error creating configuration version: %v", err)
	}

	// Create a pending run to block this run.
	r, err := b.client.Runs.Create(ctx, tfe.RunCreateOptions{
		ConfigurationVersion: c,
		Workspace:            w,
	})
	if err != nil {
		t.Fatalf("error creating pending run: %v", err)
	}

	op, configCleanup, done := testOperationPlan(t, "./testdata/plan")
	defer configCleanup()
	defer done(t)

	integrationContext := &IntegrationContext{
		B:             b,
		StopContext:   ctx,
		CancelContext: ctx,
		Op:            op,
		Run:           r,
	}

	return integrationContext, &testIntegrationOutput{
		ctx:    integrationContext,
		output: &strings.Builder{},
		t:      t,
	}
}

func TestCloud_runTasksWithTaskResults(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	integrationContext, writer := newMockIntegrationContext(b, t)

	cases := map[string]struct {
		taskResults     []*tfe.TaskResult
		context         *IntegrationContext
		writer          *testIntegrationOutput
		expectedOutputs []string
		isError         bool
	}{
		"all-succeeded": {
			taskResults: []*tfe.TaskResult{
				{ID: "1", TaskName: "Mandatory", Message: "A-OK", Status: "passed", WorkspaceTaskEnforcementLevel: "mandatory"},
				{ID: "2", TaskName: "Advisory", Message: "A-OK", Status: "passed", WorkspaceTaskEnforcementLevel: "advisory"},
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"Overall Result: Passed\n"},
			isError:         false,
		},
		"mandatory-failed": {
			taskResults: []*tfe.TaskResult{
				{ID: "1", TaskName: "Mandatory", Message: "500 Error", Status: "failed", WorkspaceTaskEnforcementLevel: "mandatory"},
				{ID: "2", TaskName: "Advisory", Message: "A-OK", Status: "passed", WorkspaceTaskEnforcementLevel: "advisory"},
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"Passed\n", "A-OK\n", "Overall Result: Failed\n"},
			isError:         true,
		},
		"advisory-failed": {
			taskResults: []*tfe.TaskResult{
				{ID: "1", TaskName: "Mandatory", Message: "A-OK", Status: "passed", WorkspaceTaskEnforcementLevel: "mandatory"},
				{ID: "2", TaskName: "Advisory", Message: "500 Error", Status: "failed", WorkspaceTaskEnforcementLevel: "advisory"},
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"Failed (Advisory)", "Overall Result: Passed with advisory failure"},
			isError:         false,
		},
		"unreachable": {
			taskResults: []*tfe.TaskResult{
				{ID: "1", TaskName: "Mandatory", Message: "", Status: "unreachable", WorkspaceTaskEnforcementLevel: "mandatory"},
				{ID: "2", TaskName: "Advisory", Message: "", Status: "unreachable", WorkspaceTaskEnforcementLevel: "advisory"},
			},
			writer:          writer,
			context:         integrationContext,
			expectedOutputs: []string{"Skipping"},
			isError:         false,
		},
	}

	for caseName, c := range cases {
		c.writer.output.Reset()
		err := b.runTasksWithTaskResults(c.context, writer, func(b *Cloud, stopCtx context.Context) (*tfe.TaskStage, error) {
			return &tfe.TaskStage{
				TaskResults: c.taskResults,
			}, nil
		})

		if c.isError && err == nil {
			t.Fatalf("Expected %s to be error", caseName)
		}

		if !c.isError && err != nil {
			t.Errorf("Expected %s to not be error but received %s", caseName, err)
		}

		output := c.writer.output.String()
		for _, expected := range c.expectedOutputs {
			if !strings.Contains(output, expected) {
				t.Fatalf("Expected output to contain '%s' but it was:\n\n%s", expected, output)
			}
		}
	}
}
