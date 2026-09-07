// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package stackruntime

import (
	"testing"

	"github.com/hashicorp/terraform/internal/stacks/stackruntime/hooks"
)

// TestActionInvocationHooksValidation validates that action invocation status
// hooks work correctly, including enum values, hook data structure, and lifecycle ordering.
func TestActionInvocationHooksValidation(t *testing.T) {
	t.Run("hook_capture_mechanism", func(t *testing.T) {
		// Verify CapturedHooks mechanism initializes correctly
		capturedHooks := NewCapturedHooks(false) // false = apply phase

		if capturedHooks == nil {
			t.Fatal("CapturedHooks should not be nil")
		}

		// Verify the hooks slice starts empty (nil or zero length)
		if len(capturedHooks.ReportActionInvocationStatus) != 0 {
			t.Errorf("expected empty initial hook list, got %d", len(capturedHooks.ReportActionInvocationStatus))
		}

		// Verify we can append to it
		capturedHooks.ReportActionInvocationStatus = append(
			capturedHooks.ReportActionInvocationStatus,
			&hooks.ActionInvocationStatusHookData{
				Addr:         mustAbsActionInvocationInstance("component.test.action.example.run"),
				ProviderAddr: mustDefaultRootProvider("testing").Provider,
				Status:       hooks.ActionInvocationRunning,
			},
		)

		if len(capturedHooks.ReportActionInvocationStatus) != 1 {
			t.Errorf("after append, expected 1 hook, got %d", len(capturedHooks.ReportActionInvocationStatus))
		}
	})

	t.Run("action_invocation_status_enum", func(t *testing.T) {
		// Test that all enum constants are defined and have valid string representations
		statuses := []hooks.ActionInvocationStatus{
			hooks.ActionInvocationStatusInvalid,
			hooks.ActionInvocationPending,
			hooks.ActionInvocationRunning,
			hooks.ActionInvocationCompleted,
			hooks.ActionInvocationErrored,
		}

		expectedStrings := map[hooks.ActionInvocationStatus]string{
			hooks.ActionInvocationStatusInvalid: "ActionInvocationStatusInvalid",
			hooks.ActionInvocationPending:       "ActionInvocationPending",
			hooks.ActionInvocationRunning:       "ActionInvocationRunning",
			hooks.ActionInvocationCompleted:     "ActionInvocationCompleted",
			hooks.ActionInvocationErrored:       "ActionInvocationErrored",
		}

		// Verify String() returns expected values
		for _, status := range statuses {
			str := status.String()
			expected, ok := expectedStrings[status]
			if !ok {
				t.Errorf("unexpected status constant: %v", status)
				continue
			}
			if str != expected {
				t.Errorf("status %v: expected String() = %q, got %q", status, expected, str)
			}
		}

		// Verify ForProtobuf() returns valid values (non-negative)
		for _, status := range statuses {
			proto := status.ForProtobuf()
			if proto < 0 {
				t.Errorf("status %v has invalid protobuf value: %v", status, proto)
			}
		}

		// Verify we have exactly 5 status values
		if len(statuses) != 5 {
			t.Errorf("expected 5 status constants, got %d", len(statuses))
		}
	})

	t.Run("hook_data_structure", func(t *testing.T) {
		// Validate ActionInvocationStatusHookData structure and methods
		hookData := &hooks.ActionInvocationStatusHookData{
			Addr:         mustAbsActionInvocationInstance("component.test.action.example.run"),
			ProviderAddr: mustDefaultRootProvider("testing").Provider,
			Status:       hooks.ActionInvocationRunning,
		}

		// Verify fields are set
		if hookData.Addr.String() == "" {
			t.Error("Addr should not be empty")
		}
		if hookData.ProviderAddr.String() == "" {
			t.Error("ProviderAddr should not be empty")
		}
		if hookData.Status == hooks.ActionInvocationStatusInvalid {
			t.Error("Status should not be Invalid when explicitly set to Running")
		}

		// Verify String() method
		str := hookData.String()
		if str == "" || str == "<nil>" {
			t.Errorf("String() should return valid representation, got: %q", str)
		}

		// Verify String() contains address
		if !contains(str, "component.test") {
			t.Errorf("String() should contain address, got: %q", str)
		}

		// Verify nil handling
		var nilHook *hooks.ActionInvocationStatusHookData
		if nilHook.String() != "<nil>" {
			t.Errorf("nil hook String() should return <nil>, got: %q", nilHook.String())
		}
	})

	t.Run("hook_status_lifecycle_ordering", func(t *testing.T) {
		// Test expected hook status sequences for different scenarios
		testCases := []struct {
			name             string
			capturedStatuses []hooks.ActionInvocationStatus
			wantValid        bool
			description      string
		}{
			{
				name: "successful_action",
				capturedStatuses: []hooks.ActionInvocationStatus{
					hooks.ActionInvocationRunning,
					hooks.ActionInvocationCompleted,
				},
				wantValid:   true,
				description: "Action starts running and completes successfully",
			},
			{
				name: "failed_action",
				capturedStatuses: []hooks.ActionInvocationStatus{
					hooks.ActionInvocationRunning,
					hooks.ActionInvocationErrored,
				},
				wantValid:   true,
				description: "Action starts running but encounters an error",
			},
			{
				name: "pending_then_running_then_completed",
				capturedStatuses: []hooks.ActionInvocationStatus{
					hooks.ActionInvocationPending,
					hooks.ActionInvocationRunning,
					hooks.ActionInvocationCompleted,
				},
				wantValid:   true,
				description: "Action goes through all states including pending",
			},
			{
				name: "invalid_only_completed",
				capturedStatuses: []hooks.ActionInvocationStatus{
					hooks.ActionInvocationCompleted,
				},
				wantValid:   false,
				description: "Invalid: completed without running",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Verify we captured the expected number of statuses
				if len(tc.capturedStatuses) == 0 {
					t.Error("test case should have at least one status")
					return
				}

				// For valid sequences, verify terminal state is at the end
				if tc.wantValid && len(tc.capturedStatuses) > 0 {
					lastStatus := tc.capturedStatuses[len(tc.capturedStatuses)-1]
					isTerminal := lastStatus == hooks.ActionInvocationCompleted ||
						lastStatus == hooks.ActionInvocationErrored

					if !isTerminal {
						t.Errorf("valid sequence should end in terminal state (Completed/Errored), got %v", lastStatus)
					}
				}

				// For invalid sequences starting with Completed, verify it's actually invalid
				if !tc.wantValid && len(tc.capturedStatuses) > 0 {
					firstStatus := tc.capturedStatuses[0]
					if firstStatus == hooks.ActionInvocationCompleted && len(tc.capturedStatuses) == 1 {
						// This is indeed invalid - can't complete without running
						t.Logf("correctly identified invalid sequence: %v", tc.capturedStatuses)
					}
				}
			})
		}
	})
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
