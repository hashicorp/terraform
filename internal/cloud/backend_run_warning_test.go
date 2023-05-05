// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package cloud

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/hashicorp/go-tfe"
	tfemocks "github.com/hashicorp/go-tfe/mocks"
	"github.com/mitchellh/cli"
)

func MockAllRunEvents(t *testing.T, client *tfe.Client) (fullRunID string, emptyRunID string) {
	ctrl := gomock.NewController(t)

	fullRunID = "run-full"
	emptyRunID = "run-empty"

	mockRunEventsAPI := tfemocks.NewMockRunEvents(ctrl)

	emptyList := tfe.RunEventList{
		Items: []*tfe.RunEvent{},
	}
	fullList := tfe.RunEventList{
		Items: []*tfe.RunEvent{
			{
				Action:      "created",
				CreatedAt:   time.Now(),
				Description: "",
			},
			{
				Action:      "changed_task_enforcements",
				CreatedAt:   time.Now(),
				Description: "The enforcement level for task 'MockTask' was changed to 'advisory' because the run task limit was exceeded.",
			},
			{
				Action:      "changed_policy_enforcements",
				CreatedAt:   time.Now(),
				Description: "The enforcement level for policy 'MockPolicy' was changed to 'advisory' because the policy limit was exceeded.",
			},
			{
				Action:      "ignored_policy_sets",
				CreatedAt:   time.Now(),
				Description: "The policy set 'MockPolicySet' was ignored because the versioned policy set limit was exceeded.",
			},
			{
				Action:      "queued",
				CreatedAt:   time.Now(),
				Description: "",
			},
		},
	}
	// Mock Full Request
	mockRunEventsAPI.
		EXPECT().
		List(gomock.Any(), fullRunID, gomock.Any()).
		Return(&fullList, nil).
		AnyTimes()

	// Mock Full Request
	mockRunEventsAPI.
		EXPECT().
		List(gomock.Any(), emptyRunID, gomock.Any()).
		Return(&emptyList, nil).
		AnyTimes()

	// Mock a bad Read response
	mockRunEventsAPI.
		EXPECT().
		List(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, tfe.ErrInvalidRunID).
		AnyTimes()

	// Wire up the mock interfaces
	client.RunEvents = mockRunEventsAPI
	return
}

func TestRunEventWarningsAll(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	config := &tfe.Config{
		Token: "not-a-token",
	}
	client, _ := tfe.NewClient(config)
	fullRunID, _ := MockAllRunEvents(t, client)

	ctx := context.TODO()

	err := b.renderRunWarnings(ctx, client, fullRunID)
	if err != nil {
		t.Fatalf("Expected to not error but received %s", err)
	}

	output := b.CLI.(*cli.MockUi).ErrorWriter.String()
	testString := "The enforcement level for task 'MockTask'"
	if !strings.Contains(output, testString) {
		t.Fatalf("Expected %q to contain %q but it did not", output, testString)
	}
	testString = "The enforcement level for policy 'MockPolicy'"
	if !strings.Contains(output, testString) {
		t.Fatalf("Expected %q to contain %q but it did not", output, testString)
	}
	testString = "The policy set 'MockPolicySet'"
	if !strings.Contains(output, testString) {
		t.Fatalf("Expected %q to contain %q but it did not", output, testString)
	}
}

func TestRunEventWarningsEmpty(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	config := &tfe.Config{
		Token: "not-a-token",
	}
	client, _ := tfe.NewClient(config)
	_, emptyRunID := MockAllRunEvents(t, client)

	ctx := context.TODO()

	err := b.renderRunWarnings(ctx, client, emptyRunID)
	if err != nil {
		t.Fatalf("Expected to not error but received %s", err)
	}

	output := b.CLI.(*cli.MockUi).ErrorWriter.String()
	if output != "" {
		t.Fatalf("Expected %q to be empty but it was not", output)
	}
}

func TestRunEventWarningsWithError(t *testing.T) {
	b, bCleanup := testBackendWithName(t)
	defer bCleanup()

	config := &tfe.Config{
		Token: "not-a-token",
	}
	client, _ := tfe.NewClient(config)
	MockAllRunEvents(t, client)

	ctx := context.TODO()

	err := b.renderRunWarnings(ctx, client, "bad run id")

	if err == nil {
		t.Error("Expected to error but did not")
	}
}
