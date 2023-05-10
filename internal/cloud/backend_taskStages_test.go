// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-tfe"
	tfemocks "github.com/hashicorp/go-tfe/mocks"
)

func MockAllTaskStages(t *testing.T, client *tfe.Client) (RunID string) {
	ctrl := gomock.NewController(t)

	RunID = "run-all_task_stages"

	mockRunsAPI := tfemocks.NewMockRuns(ctrl)

	goodRun := tfe.Run{
		TaskStages: []*tfe.TaskStage{
			{
				Stage: tfe.PrePlan,
			},
			{
				Stage: tfe.PostPlan,
			},
			{
				Stage: tfe.PreApply,
			},
		},
	}
	mockRunsAPI.
		EXPECT().
		ReadWithOptions(gomock.Any(), RunID, gomock.Any()).
		Return(&goodRun, nil).
		AnyTimes()

	// Mock a bad Read response
	mockRunsAPI.
		EXPECT().
		ReadWithOptions(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, tfe.ErrInvalidOrg).
		AnyTimes()

	// Wire up the mock interfaces
	client.Runs = mockRunsAPI
	return
}

func MockPrePlanTaskStage(t *testing.T, client *tfe.Client) (RunID string) {
	ctrl := gomock.NewController(t)

	RunID = "run-pre_plan_task_stage"

	mockRunsAPI := tfemocks.NewMockRuns(ctrl)

	goodRun := tfe.Run{
		TaskStages: []*tfe.TaskStage{
			{
				Stage: tfe.PrePlan,
			},
		},
	}
	mockRunsAPI.
		EXPECT().
		ReadWithOptions(gomock.Any(), RunID, gomock.Any()).
		Return(&goodRun, nil).
		AnyTimes()

	// Mock a bad Read response
	mockRunsAPI.
		EXPECT().
		ReadWithOptions(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, tfe.ErrInvalidOrg).
		AnyTimes()

	// Wire up the mock interfaces
	client.Runs = mockRunsAPI
	return
}

func MockTaskStageUnsupported(t *testing.T, client *tfe.Client) (RunID string) {
	ctrl := gomock.NewController(t)

	RunID = "run-unsupported_task_stage"

	mockRunsAPI := tfemocks.NewMockRuns(ctrl)

	mockRunsAPI.
		EXPECT().
		ReadWithOptions(gomock.Any(), RunID, gomock.Any()).
		Return(nil, errors.New("Invalid include parameter")).
		AnyTimes()

	mockRunsAPI.
		EXPECT().
		ReadWithOptions(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, tfe.ErrInvalidOrg).
		AnyTimes()

	client.Runs = mockRunsAPI
	return
}

func TestTaskStagesWithAllStages(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	config := &tfe.Config{
		Token: "not-a-token",
	}
	client, _ := tfe.NewClient(config)
	runID := MockAllTaskStages(t, client)

	ctx := context.TODO()
	taskStages, err := b.runTaskStages(ctx, client, runID)

	if err != nil {
		t.Fatalf("Expected to not error but received %s", err)
	}

	for _, stageName := range []tfe.Stage{
		tfe.PrePlan,
		tfe.PostPlan,
		tfe.PreApply,
	} {
		if stage, ok := taskStages[stageName]; ok {
			if stage.Stage != stageName {
				t.Errorf("Expected task stage indexed by %s to find a Task Stage with the same index, but receieved %s", stageName, stage.Stage)
			}
		} else {
			t.Errorf("Expected task stage indexed by %s to exist, but it did not", stageName)
		}
	}
}

func TestTaskStagesWithOneStage(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	config := &tfe.Config{
		Token: "not-a-token",
	}
	client, _ := tfe.NewClient(config)
	runID := MockPrePlanTaskStage(t, client)

	ctx := context.TODO()
	taskStages, err := b.runTaskStages(ctx, client, runID)

	if err != nil {
		t.Fatalf("Expected to not error but received %s", err)
	}

	if _, ok := taskStages[tfe.PrePlan]; !ok {
		t.Errorf("Expected task stage indexed by %s to exist, but it did not", tfe.PrePlan)
	}

	for _, stageName := range []tfe.Stage{
		tfe.PostPlan,
		tfe.PreApply,
	} {
		if _, ok := taskStages[stageName]; ok {
			t.Errorf("Expected task stage indexed by %s to not exist, but it did", stageName)
		}
	}
}

func TestTaskStagesWithOldTFC(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	config := &tfe.Config{
		Token: "not-a-token",
	}
	client, _ := tfe.NewClient(config)
	runID := MockTaskStageUnsupported(t, client)

	ctx := context.TODO()
	taskStages, err := b.runTaskStages(ctx, client, runID)

	if err != nil {
		t.Fatalf("Expected to not error but received %s", err)
	}

	if len(taskStages) != 0 {
		t.Errorf("Expected task stage to be empty, but found %d stages", len(taskStages))
	}
}

func TestTaskStagesWithErrors(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	config := &tfe.Config{
		Token: "not-a-token",
	}
	client, _ := tfe.NewClient(config)
	MockTaskStageUnsupported(t, client)

	ctx := context.TODO()
	_, err := b.runTaskStages(ctx, client, "this run ID will not exist is invalid anyway")

	if err == nil {
		t.Error("Expected to error but did not")
	}
}

func TestTaskStageOverride(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	integrationContext, writer := newMockIntegrationContext(b, t)

	integrationContext.Op.UIOut = b.CLI

	cases := map[string]struct {
		taskStageID string
		isError     bool
		errMsg      string
		input       *mockInput
		cont        bool
	}{
		"override-pass": {
			taskStageID: "ts-pass",
			isError:     false,
			input: testInput(t, map[string]string{
				"→→ [bold]Override": "override",
			}),
			errMsg: "",
			cont:   true,
		},
		"override-fail": {
			taskStageID: "ts-err",
			isError:     true,
			input: testInput(t, map[string]string{
				"→→ [bold]Override": "override",
			}),
			errMsg: "",
			cont:   false,
		},
		"skip-override": {
			taskStageID: "ts-err",
			isError:     true,
			errMsg:      "Failed to override: Apply discarded.",
			input: testInput(t, map[string]string{
				"→→ [bold]Override": "no",
			}),
			cont: false,
		},
	}
	for _, c := range cases {
		integrationContext.Op.UIIn = c.input
		cont, err := b.processStageOverrides(integrationContext, writer, c.taskStageID)
		if c.isError {
			if err == nil {
				t.Fatalf("Expected to fail with some error")
			}
			if c.errMsg != "" {
				if !strings.Contains(err.Error(), c.errMsg) {
					t.Fatalf("Expected: %s, got: %s", c.errMsg, err.Error())
				}
			}

		} else {
			if err != nil {
				t.Fatalf("Expected to pass, got err: %s", err)
			}
		}
		if c.cont != cont {
			t.Fatalf("expected polling continue: %t, got: %t", c.cont, cont)
		}
	}
}
