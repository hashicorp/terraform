// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terminal"
	"github.com/hashicorp/terraform/internal/terraform"
)

func testJSONHookResourceID(addr addrs.AbsResourceInstance) terraform.HookResourceIdentity {
	return terraform.HookResourceIdentity{
		Addr: addr,
		ProviderAddr: addrs.Provider{
			Type:      "test",
			Namespace: "hashicorp",
			Hostname:  "example.com",
		},
	}
}

func testJSONHookActionID(actionAddr addrs.AbsActionInstance, triggeringResourceAddr addrs.AbsResourceInstance, actionTriggerIndex int, actionsListIndex int) terraform.HookActionIdentity {
	return terraform.HookActionIdentity{
		Addr: actionAddr,
		ActionTrigger: &plans.LifecycleActionTrigger{
			TriggeringResourceAddr:  triggeringResourceAddr,
			ActionTriggerBlockIndex: actionTriggerIndex,
			ActionsListIndex:        actionsListIndex,
			ActionTriggerEvent:      configs.AfterCreate,
		},
	}
}

// Test a sequence of hooks associated with creating a resource
func TestJSONHook_create(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	hook := newJSONHook(NewJSONView(NewView(streams)))

	var nowMu sync.Mutex
	now := time.Now()
	hook.timeNow = func() time.Time {
		nowMu.Lock()
		defer nowMu.Unlock()
		return now
	}

	after := make(chan time.Time, 1)
	hook.timeAfter = func(time.Duration) <-chan time.Time { return after }

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "boop",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)
	priorState := cty.NullVal(cty.Object(map[string]cty.Type{
		"id":  cty.String,
		"bar": cty.List(cty.String),
	}))
	plannedNewState := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("test"),
		"bar": cty.ListVal([]cty.Value{
			cty.StringVal("baz"),
		}),
	})

	action, err := hook.PreApply(testJSONHookResourceID(addr), addrs.NotDeposed, plans.Create, priorState, plannedNewState)
	testHookReturnValues(t, action, err)

	action, err = hook.PreProvisionInstanceStep(testJSONHookResourceID(addr), "local-exec")
	testHookReturnValues(t, action, err)

	hook.ProvisionOutput(testJSONHookResourceID(addr), "local-exec", `Executing: ["/bin/sh" "-c" "touch /etc/motd"]`)

	action, err = hook.PostProvisionInstanceStep(testJSONHookResourceID(addr), "local-exec", nil)
	testHookReturnValues(t, action, err)

	// Travel 10s into the future, notify the progress goroutine, and sleep
	// briefly to allow it to execute
	nowMu.Lock()
	now = now.Add(10 * time.Second)
	after <- now
	nowMu.Unlock()
	time.Sleep(10 * time.Millisecond)

	// Travel 10s into the future, notify the progress goroutine, and sleep
	// briefly to allow it to execute
	nowMu.Lock()
	now = now.Add(10 * time.Second)
	after <- now
	nowMu.Unlock()
	time.Sleep(10 * time.Millisecond)

	// Travel 2s into the future. We have arrived!
	nowMu.Lock()
	now = now.Add(2 * time.Second)
	nowMu.Unlock()

	action, err = hook.PostApply(testJSONHookResourceID(addr), addrs.NotDeposed, plannedNewState, nil)
	testHookReturnValues(t, action, err)

	// Shut down the progress goroutine if still active
	hook.resourceProgressMu.Lock()
	for key, progress := range hook.resourceProgress {
		close(progress.done)
		<-progress.heartbeatDone
		delete(hook.resourceProgress, key)
	}
	hook.resourceProgressMu.Unlock()

	wantResource := map[string]interface{}{
		"addr":             string("test_instance.boop"),
		"implied_provider": string("test"),
		"module":           string(""),
		"resource":         string("test_instance.boop"),
		"resource_key":     nil,
		"resource_name":    string("boop"),
		"resource_type":    string("test_instance"),
	}
	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "test_instance.boop: Creating...",
			"@module":  "terraform.ui",
			"type":     "apply_start",
			"hook": map[string]interface{}{
				"action":   string("create"),
				"resource": wantResource,
			},
		},
		{
			"@level":   "info",
			"@message": "test_instance.boop: Provisioning with 'local-exec'...",
			"@module":  "terraform.ui",
			"type":     "provision_start",
			"hook": map[string]interface{}{
				"provisioner": "local-exec",
				"resource":    wantResource,
			},
		},
		{
			"@level":   "info",
			"@message": `test_instance.boop: (local-exec): Executing: ["/bin/sh" "-c" "touch /etc/motd"]`,
			"@module":  "terraform.ui",
			"type":     "provision_progress",
			"hook": map[string]interface{}{
				"output":      `Executing: ["/bin/sh" "-c" "touch /etc/motd"]`,
				"provisioner": "local-exec",
				"resource":    wantResource,
			},
		},
		{
			"@level":   "info",
			"@message": "test_instance.boop: (local-exec) Provisioning complete",
			"@module":  "terraform.ui",
			"type":     "provision_complete",
			"hook": map[string]interface{}{
				"provisioner": "local-exec",
				"resource":    wantResource,
			},
		},
		{
			"@level":   "info",
			"@message": "test_instance.boop: Still creating... [10s elapsed]",
			"@module":  "terraform.ui",
			"type":     "apply_progress",
			"hook": map[string]interface{}{
				"action":          string("create"),
				"elapsed_seconds": float64(10),
				"resource":        wantResource,
			},
		},
		{
			"@level":   "info",
			"@message": "test_instance.boop: Still creating... [20s elapsed]",
			"@module":  "terraform.ui",
			"type":     "apply_progress",
			"hook": map[string]interface{}{
				"action":          string("create"),
				"elapsed_seconds": float64(20),
				"resource":        wantResource,
			},
		},
		{
			"@level":   "info",
			"@message": "test_instance.boop: Creation complete after 22s [id=test]",
			"@module":  "terraform.ui",
			"type":     "apply_complete",
			"hook": map[string]interface{}{
				"action":          string("create"),
				"elapsed_seconds": float64(22),
				"id_key":          "id",
				"id_value":        "test",
				"resource":        wantResource,
			},
		},
	}

	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestJSONHook_errors(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	hook := newJSONHook(NewJSONView(NewView(streams)))

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "boop",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)
	priorState := cty.NullVal(cty.Object(map[string]cty.Type{
		"id":  cty.String,
		"bar": cty.List(cty.String),
	}))
	plannedNewState := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("test"),
		"bar": cty.ListVal([]cty.Value{
			cty.StringVal("baz"),
		}),
	})

	action, err := hook.PreApply(testJSONHookResourceID(addr), addrs.NotDeposed, plans.Delete, priorState, plannedNewState)
	testHookReturnValues(t, action, err)

	provisionError := fmt.Errorf("provisioner didn't want to")
	action, err = hook.PostProvisionInstanceStep(testJSONHookResourceID(addr), "local-exec", provisionError)
	testHookReturnValues(t, action, err)

	applyError := fmt.Errorf("provider was sad")
	action, err = hook.PostApply(testJSONHookResourceID(addr), addrs.NotDeposed, plannedNewState, applyError)
	testHookReturnValues(t, action, err)

	// Shut down the progress goroutine
	hook.resourceProgressMu.Lock()
	for key, progress := range hook.resourceProgress {
		close(progress.done)
		<-progress.heartbeatDone
		delete(hook.resourceProgress, key)
	}
	hook.resourceProgressMu.Unlock()

	wantResource := map[string]interface{}{
		"addr":             string("test_instance.boop"),
		"implied_provider": string("test"),
		"module":           string(""),
		"resource":         string("test_instance.boop"),
		"resource_key":     nil,
		"resource_name":    string("boop"),
		"resource_type":    string("test_instance"),
	}
	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "test_instance.boop: Destroying...",
			"@module":  "terraform.ui",
			"type":     "apply_start",
			"hook": map[string]interface{}{
				"action":   string("delete"),
				"resource": wantResource,
			},
		},
		{
			"@level":   "info",
			"@message": "test_instance.boop: (local-exec) Provisioning errored",
			"@module":  "terraform.ui",
			"type":     "provision_errored",
			"hook": map[string]interface{}{
				"provisioner": "local-exec",
				"resource":    wantResource,
			},
		},
		{
			"@level":   "info",
			"@message": "test_instance.boop: Destruction errored after 0s",
			"@module":  "terraform.ui",
			"type":     "apply_errored",
			"hook": map[string]interface{}{
				"action":          string("delete"),
				"elapsed_seconds": float64(0),
				"resource":        wantResource,
			},
		},
	}

	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestJSONHook_refresh(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	hook := newJSONHook(NewJSONView(NewView(streams)))

	addr := addrs.Resource{
		Mode: addrs.DataResourceMode,
		Type: "test_data_source",
		Name: "beep",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)
	state := cty.ObjectVal(map[string]cty.Value{
		"id": cty.StringVal("honk"),
		"bar": cty.ListVal([]cty.Value{
			cty.StringVal("baz"),
		}),
	})

	action, err := hook.PreRefresh(testJSONHookResourceID(addr), addrs.NotDeposed, state)
	testHookReturnValues(t, action, err)

	action, err = hook.PostRefresh(testJSONHookResourceID(addr), addrs.NotDeposed, state, state)
	testHookReturnValues(t, action, err)

	wantResource := map[string]interface{}{
		"addr":             string("data.test_data_source.beep"),
		"implied_provider": string("test"),
		"module":           string(""),
		"resource":         string("data.test_data_source.beep"),
		"resource_key":     nil,
		"resource_name":    string("beep"),
		"resource_type":    string("test_data_source"),
	}
	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "data.test_data_source.beep: Refreshing state... [id=honk]",
			"@module":  "terraform.ui",
			"type":     "refresh_start",
			"hook": map[string]interface{}{
				"resource": wantResource,
				"id_key":   "id",
				"id_value": "honk",
			},
		},
		{
			"@level":   "info",
			"@message": "data.test_data_source.beep: Refresh complete [id=honk]",
			"@module":  "terraform.ui",
			"type":     "refresh_complete",
			"hook": map[string]interface{}{
				"resource": wantResource,
				"id_key":   "id",
				"id_value": "honk",
			},
		},
	}

	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestJSONHook_EphemeralOp(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	hook := newJSONHook(NewJSONView(NewView(streams)))

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "boop",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	action, err := hook.PreEphemeralOp(testJSONHookResourceID(addr), plans.Open)
	testHookReturnValues(t, action, err)

	action, err = hook.PostEphemeralOp(testJSONHookResourceID(addr), plans.Open, nil)
	testHookReturnValues(t, action, err)

	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "test_instance.boop: Opening...",
			"@module":  "terraform.ui",
			"type":     "ephemeral_op_start",
			"hook": map[string]interface{}{
				"action": string("open"),
				"resource": map[string]interface{}{
					"addr":             string("test_instance.boop"),
					"implied_provider": string("test"),
					"module":           string(""),
					"resource":         string("test_instance.boop"),
					"resource_key":     nil,
					"resource_name":    string("boop"),
					"resource_type":    string("test_instance"),
				},
			},
		},
		{
			"@level":   "info",
			"@message": "test_instance.boop: Opening complete after 0s",
			"@module":  "terraform.ui",
			"type":     "ephemeral_op_complete",
			"hook": map[string]interface{}{
				"action":          string("open"),
				"elapsed_seconds": float64(0),
				"resource": map[string]interface{}{
					"addr":             string("test_instance.boop"),
					"implied_provider": string("test"),
					"module":           string(""),
					"resource":         string("test_instance.boop"),
					"resource_key":     nil,
					"resource_name":    string("boop"),
					"resource_type":    string("test_instance"),
				},
			},
		},
	}

	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestJSONHook_EphemeralOp_progress(t *testing.T) {
	syncTest, streams, done := streamableSyncTest(t)

	syncTest(t, func(t *testing.T) {
		start := time.Now()
		hook := newJSONHook(NewJSONView(NewView(streams)))
		hook.periodicUiTimer = 1 * time.Second
		t.Log(time.Since(start))

		addr := addrs.Resource{
			Mode: addrs.ManagedResourceMode,
			Type: "test_instance",
			Name: "boop",
		}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

		action, err := hook.PreEphemeralOp(testJSONHookResourceID(addr), plans.Open)
		testHookReturnValues(t, action, err)
		time.Sleep(2005 * time.Millisecond)

		action, err = hook.PostEphemeralOp(testJSONHookResourceID(addr), plans.Open, nil)
		testHookReturnValues(t, action, err)

		want := []map[string]interface{}{
			{
				"@level":   "info",
				"@message": "test_instance.boop: Opening...",
				"@module":  "terraform.ui",
				"type":     "ephemeral_op_start",
				"hook": map[string]interface{}{
					"action": string("open"),
					"resource": map[string]interface{}{
						"addr":             string("test_instance.boop"),
						"implied_provider": string("test"),
						"module":           string(""),
						"resource":         string("test_instance.boop"),
						"resource_key":     nil,
						"resource_name":    string("boop"),
						"resource_type":    string("test_instance"),
					},
				},
			},
			{
				"@level":   "info",
				"@message": "test_instance.boop: Still opening... [1s elapsed]",
				"@module":  "terraform.ui",
				"type":     "ephemeral_op_progress",
				"hook": map[string]interface{}{
					"action":          string("open"),
					"elapsed_seconds": float64(1),
					"resource": map[string]interface{}{
						"addr":             string("test_instance.boop"),
						"implied_provider": string("test"),
						"module":           string(""),
						"resource":         string("test_instance.boop"),
						"resource_key":     nil,
						"resource_name":    string("boop"),
						"resource_type":    string("test_instance"),
					},
				},
			},
			{
				"@level":   "info",
				"@message": "test_instance.boop: Still opening... [2s elapsed]",
				"@module":  "terraform.ui",
				"type":     "ephemeral_op_progress",
				"hook": map[string]interface{}{
					"action":          string("open"),
					"elapsed_seconds": float64(2),
					"resource": map[string]interface{}{
						"addr":             string("test_instance.boop"),
						"implied_provider": string("test"),
						"module":           string(""),
						"resource":         string("test_instance.boop"),
						"resource_key":     nil,
						"resource_name":    string("boop"),
						"resource_type":    string("test_instance"),
					},
				},
			},
			{
				"@level":   "info",
				"@message": "test_instance.boop: Opening complete after 2s",
				"@module":  "terraform.ui",
				"type":     "ephemeral_op_complete",
				"hook": map[string]interface{}{
					"action":          string("open"),
					"elapsed_seconds": float64(2),
					"resource": map[string]interface{}{
						"addr":             string("test_instance.boop"),
						"implied_provider": string("test"),
						"module":           string(""),
						"resource":         string("test_instance.boop"),
						"resource_key":     nil,
						"resource_name":    string("boop"),
						"resource_type":    string("test_instance"),
					},
				},
			},
		}
		stdout := done(t).Stdout()
		testJSONViewOutputEquals(t, stdout, want)
	})
}

func TestJSONHook_EphemeralOp_error(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	hook := newJSONHook(NewJSONView(NewView(streams)))

	addr := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "boop",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	action, err := hook.PreEphemeralOp(testJSONHookResourceID(addr), plans.Open)
	testHookReturnValues(t, action, err)

	action, err = hook.PostEphemeralOp(testJSONHookResourceID(addr), plans.Open, errors.New("test error"))
	testHookReturnValues(t, action, err)

	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "test_instance.boop: Opening...",
			"@module":  "terraform.ui",
			"type":     "ephemeral_op_start",
			"hook": map[string]interface{}{
				"action": string("open"),
				"resource": map[string]interface{}{
					"addr":             string("test_instance.boop"),
					"implied_provider": string("test"),
					"module":           string(""),
					"resource":         string("test_instance.boop"),
					"resource_key":     nil,
					"resource_name":    string("boop"),
					"resource_type":    string("test_instance"),
				},
			},
		},
		{
			"@level":   "info",
			"@message": "test_instance.boop: Opening errored after 0s",
			"@module":  "terraform.ui",
			"type":     "ephemeral_op_errored",
			"hook": map[string]interface{}{
				"action":          string("open"),
				"elapsed_seconds": float64(0),
				"resource": map[string]interface{}{
					"addr":             string("test_instance.boop"),
					"implied_provider": string("test"),
					"module":           string(""),
					"resource":         string("test_instance.boop"),
					"resource_key":     nil,
					"resource_name":    string("boop"),
					"resource_type":    string("test_instance"),
				},
			},
		},
	}

	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func TestJSONHook_actions(t *testing.T) {
	streams, done := terminal.StreamsForTesting(t)
	hook := newJSONHook(NewJSONView(NewView(streams)))

	actionA := addrs.AbsActionInstance{
		Module: addrs.RootModuleInstance,
		Action: addrs.Action{
			Type: "aws_lambda_invocation",
			Name: "notify_slack",
		}.Instance(addrs.IntKey(42)),
	}

	resourceA := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "boop",
	}.Instance(addrs.NoKey).Absolute(addrs.RootModuleInstance)

	subModule := addrs.RootModuleInstance.Child("childMod", addrs.StringKey("infra"))
	actionB := addrs.AbsActionInstance{
		Module: subModule,
		Action: addrs.Action{
			Type: "ansible_playbook",
			Name: "webserver",
		}.Instance(addrs.NoKey),
	}

	resourceB := addrs.Resource{
		Mode: addrs.ManagedResourceMode,
		Type: "test_instance",
		Name: "boop",
	}.Instance(addrs.NoKey).Absolute(subModule)

	actionAHookId := testJSONHookActionID(actionA, resourceA, 0, 1)
	actionBHookId := testJSONHookActionID(actionB, resourceB, 2, 3)

	action, err := hook.StartAction(actionAHookId)
	testHookReturnValues(t, action, err)

	action, err = hook.ProgressAction(actionAHookId, "Hello world from the lambda function")
	testHookReturnValues(t, action, err)

	action, err = hook.StartAction(actionBHookId)
	testHookReturnValues(t, action, err)

	action, err = hook.ProgressAction(actionBHookId, "TASK: [hello]\n ok: [localhost] => (item=Hello world from the ansible playbook]")
	testHookReturnValues(t, action, err)

	action, err = hook.CompleteAction(actionBHookId, nil)
	testHookReturnValues(t, action, err)

	action, err = hook.CompleteAction(actionAHookId, errors.New("lambda terminated with exit code 1"))
	testHookReturnValues(t, action, err)

	want := []map[string]interface{}{
		{
			"@level":   "info",
			"@message": "test_instance.boop.trigger[0]: Action Started: action.aws_lambda_invocation.notify_slack[42]",
			"@module":  "terraform.ui",
			"type":     "action_start",
			"hook": map[string]interface{}{
				"action": map[string]interface{}{
					"addr":             "action.aws_lambda_invocation.notify_slack[42]",
					"module":           "",
					"implied_provider": "aws",
					"resource":         "action.aws_lambda_invocation.notify_slack[42]",
					"resource_key":     float64(42),
					"resource_name":    "notify_slack",
					"resource_type":    "aws_lambda_invocation",
				},
				"actions_index": float64(1),
				"resource": map[string]interface{}{
					"addr":             "test_instance.boop",
					"implied_provider": "test",
					"module":           "",
					"resource":         "test_instance.boop",
					"resource_key":     nil,
					"resource_name":    "boop",
					"resource_type":    "test_instance",
				},
				"trigger_event": "AfterCreate",
				"trigger_index": float64(0),
			},
		},
		{
			"@level":   "info",
			"@message": "test_instance.boop (0): action.aws_lambda_invocation.notify_slack[42] - Hello world from the lambda function",
			"@module":  "terraform.ui",
			"type":     "action_progress",
			"hook": map[string]interface{}{
				"action": map[string]interface{}{
					"addr":             "action.aws_lambda_invocation.notify_slack[42]",
					"module":           "",
					"implied_provider": "aws",
					"resource":         "action.aws_lambda_invocation.notify_slack[42]",
					"resource_key":     float64(42),
					"resource_name":    "notify_slack",
					"resource_type":    "aws_lambda_invocation",
				},
				"actions_index": float64(1),
				"message":       "Hello world from the lambda function",
				"resource": map[string]interface{}{
					"addr":             "test_instance.boop",
					"implied_provider": "test",
					"module":           "",
					"resource":         "test_instance.boop",
					"resource_key":     nil,
					"resource_name":    "boop",
					"resource_type":    "test_instance",
				},
				"trigger_event": "AfterCreate",
				"trigger_index": float64(0),
			},
		},
		{
			"@level":   "info",
			"@message": "module.childMod[\"infra\"].test_instance.boop.trigger[2]: Action Started: module.childMod[\"infra\"].action.ansible_playbook.webserver",
			"@module":  "terraform.ui",
			"type":     "action_start",
			"hook": map[string]interface{}{
				"action": map[string]interface{}{
					"addr":             "module.childMod[\"infra\"].action.ansible_playbook.webserver",
					"module":           "module.childMod[\"infra\"]",
					"implied_provider": "ansible",
					"resource":         "action.ansible_playbook.webserver",
					"resource_key":     nil,
					"resource_name":    "webserver",
					"resource_type":    "ansible_playbook",
				},
				"actions_index": float64(3),
				"resource": map[string]interface{}{
					"addr":             "module.childMod[\"infra\"].test_instance.boop",
					"implied_provider": "test",
					"module":           "module.childMod[\"infra\"]",
					"resource":         "test_instance.boop",
					"resource_key":     nil,
					"resource_name":    "boop",
					"resource_type":    "test_instance",
				},
				"trigger_event": "AfterCreate",
				"trigger_index": float64(2),
			},
		},
		{
			"@level":   "info",
			"@message": "module.childMod[\"infra\"].test_instance.boop (2): module.childMod[\"infra\"].action.ansible_playbook.webserver - TASK: [hello]\n ok: [localhost] => (item=Hello world from the ansible playbook]",
			"@module":  "terraform.ui",
			"type":     "action_progress",
			"hook": map[string]interface{}{
				"action": map[string]interface{}{
					"addr":             "module.childMod[\"infra\"].action.ansible_playbook.webserver",
					"module":           "module.childMod[\"infra\"]",
					"implied_provider": "ansible",
					"resource":         "action.ansible_playbook.webserver",
					"resource_key":     nil,
					"resource_name":    "webserver",
					"resource_type":    "ansible_playbook",
				},
				"actions_index": float64(3),
				"message":       "TASK: [hello]\n ok: [localhost] => (item=Hello world from the ansible playbook]",
				"resource": map[string]interface{}{
					"addr":             "module.childMod[\"infra\"].test_instance.boop",
					"implied_provider": "test",
					"module":           "module.childMod[\"infra\"]",
					"resource":         "test_instance.boop",
					"resource_key":     nil,
					"resource_name":    "boop",
					"resource_type":    "test_instance",
				},
				"trigger_event": "AfterCreate",
				"trigger_index": float64(2),
			},
		},
		{
			"@level":   "info",
			"@message": "module.childMod[\"infra\"].test_instance.boop (2): Action Complete: module.childMod[\"infra\"].action.ansible_playbook.webserver",
			"@module":  "terraform.ui",
			"type":     "action_complete",
			"hook": map[string]interface{}{
				"action": map[string]interface{}{
					"addr":             "module.childMod[\"infra\"].action.ansible_playbook.webserver",
					"module":           "module.childMod[\"infra\"]",
					"implied_provider": "ansible",
					"resource":         "action.ansible_playbook.webserver",
					"resource_key":     nil,
					"resource_name":    "webserver",
					"resource_type":    "ansible_playbook",
				},
				"actions_index": float64(3),
				"resource": map[string]interface{}{
					"addr":             "module.childMod[\"infra\"].test_instance.boop",
					"implied_provider": "test",
					"module":           "module.childMod[\"infra\"]",
					"resource":         "test_instance.boop",
					"resource_key":     nil,
					"resource_name":    "boop",
					"resource_type":    "test_instance",
				},
				"trigger_event": "AfterCreate",
				"trigger_index": float64(2),
			},
		},
		{
			"@level":   "info",
			"@message": "test_instance.boop (0): Action Errored: action.aws_lambda_invocation.notify_slack[42] - lambda terminated with exit code 1",
			"@module":  "terraform.ui",
			"type":     "action_errored",
			"hook": map[string]interface{}{
				"action": map[string]interface{}{
					"addr":             "action.aws_lambda_invocation.notify_slack[42]",
					"module":           "",
					"implied_provider": "aws",
					"resource":         "action.aws_lambda_invocation.notify_slack[42]",
					"resource_key":     float64(42),
					"resource_name":    "notify_slack",
					"resource_type":    "aws_lambda_invocation",
				},
				"actions_index": float64(1),
				"error":         "lambda terminated with exit code 1",
				"resource": map[string]interface{}{
					"addr":             "test_instance.boop",
					"implied_provider": "test",
					"module":           "",
					"resource":         "test_instance.boop",
					"resource_key":     nil,
					"resource_name":    "boop",
					"resource_type":    "test_instance",
				},
				"trigger_event": "AfterCreate",
				"trigger_index": float64(0),
			},
		},
	}

	testJSONViewOutputEquals(t, done(t).Stdout(), want)
}

func testHookReturnValues(t *testing.T, action terraform.HookAction, err error) {
	t.Helper()

	if err != nil {
		t.Fatal(err)
	}
	if action != terraform.HookActionContinue {
		t.Fatalf("Expected hook to continue, given: %#v", action)
	}
}

// streamableSyncTest is a helper to ensure that the long-running streaming goroutines are started outside of the synctest bubble.
// Otherwise, the sync bubble will be unable to advance time, and the main goroutine will become infinitely paused on any time.Sleep operation.
func streamableSyncTest(t *testing.T) (func(t *testing.T, f func(*testing.T)), *terminal.Streams, func(*testing.T) *terminal.TestOutput) {
	streams, done := terminal.StreamsForTesting(t)
	return synctest.Test, streams, done
}
