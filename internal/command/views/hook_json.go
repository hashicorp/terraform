// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package views

import (
	"bufio"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/command/format"
	"github.com/hashicorp/terraform/internal/command/views/json"
	"github.com/hashicorp/terraform/internal/plans"
	"github.com/hashicorp/terraform/internal/terraform"
)

// How long to wait between sending heartbeat/progress messages
const heartbeatInterval = 10 * time.Second

func newJSONHook(view *JSONView) *jsonHook {
	return &jsonHook{
		view:      view,
		applying:  make(map[string]applyProgress),
		timeNow:   time.Now,
		timeAfter: time.After,
	}
}

type jsonHook struct {
	terraform.NilHook

	view *JSONView

	// Concurrent map of resource addresses to allow the sequence of pre-apply,
	// progress, and post-apply messages to share data about the resource
	applying     map[string]applyProgress
	applyingLock sync.Mutex

	// Mockable functions for testing the progress timer goroutine
	timeNow   func() time.Time
	timeAfter func(time.Duration) <-chan time.Time
}

var _ terraform.Hook = (*jsonHook)(nil)

type applyProgress struct {
	addr   addrs.AbsResourceInstance
	action plans.Action
	start  time.Time

	// done is used for post-apply to stop the progress goroutine
	done chan struct{}

	// heartbeatDone is used to allow tests to safely wait for the progress
	// goroutine to finish
	heartbeatDone chan struct{}
}

func (h *jsonHook) PreApply(id terraform.HookResourceIdentity, dk addrs.DeposedKey, action plans.Action, priorState, plannedNewState cty.Value) (terraform.HookAction, error) {
	if action != plans.NoOp {
		idKey, idValue := format.ObjectValueIDOrName(priorState)
		h.view.Hook(json.NewApplyStart(id.Addr, action, idKey, idValue))
	}

	progress := applyProgress{
		addr:          id.Addr,
		action:        action,
		start:         h.timeNow().Round(time.Second),
		done:          make(chan struct{}),
		heartbeatDone: make(chan struct{}),
	}
	h.applyingLock.Lock()
	h.applying[id.Addr.String()] = progress
	h.applyingLock.Unlock()

	if action != plans.NoOp {
		go h.applyingHeartbeat(progress)
	}
	return terraform.HookActionContinue, nil
}

func (h *jsonHook) applyingHeartbeat(progress applyProgress) {
	defer close(progress.heartbeatDone)
	for {
		select {
		case <-progress.done:
			return
		case <-h.timeAfter(heartbeatInterval):
		}

		elapsed := h.timeNow().Round(time.Second).Sub(progress.start)
		h.view.Hook(json.NewApplyProgress(progress.addr, progress.action, elapsed))
	}
}

func (h *jsonHook) PostApply(id terraform.HookResourceIdentity, dk addrs.DeposedKey, newState cty.Value, err error) (terraform.HookAction, error) {
	key := id.Addr.String()
	h.applyingLock.Lock()
	progress := h.applying[key]
	if progress.done != nil {
		close(progress.done)
	}
	delete(h.applying, key)
	h.applyingLock.Unlock()

	if progress.action == plans.NoOp {
		return terraform.HookActionContinue, nil
	}

	elapsed := h.timeNow().Round(time.Second).Sub(progress.start)

	if err != nil {
		// Errors are collected and displayed post-apply, so no need to
		// re-render them here. Instead just signal that this resource failed
		// to apply.
		h.view.Hook(json.NewApplyErrored(id.Addr, progress.action, elapsed))
	} else {
		idKey, idValue := format.ObjectValueID(newState)
		h.view.Hook(json.NewApplyComplete(id.Addr, progress.action, idKey, idValue, elapsed))
	}
	return terraform.HookActionContinue, nil
}

func (h *jsonHook) PreProvisionInstanceStep(id terraform.HookResourceIdentity, typeName string) (terraform.HookAction, error) {
	h.view.Hook(json.NewProvisionStart(id.Addr, typeName))
	return terraform.HookActionContinue, nil
}

func (h *jsonHook) PostProvisionInstanceStep(id terraform.HookResourceIdentity, typeName string, err error) (terraform.HookAction, error) {
	if err != nil {
		// Errors are collected and displayed post-apply, so no need to
		// re-render them here. Instead just signal that this provisioner step
		// failed.
		h.view.Hook(json.NewProvisionErrored(id.Addr, typeName))
	} else {
		h.view.Hook(json.NewProvisionComplete(id.Addr, typeName))
	}
	return terraform.HookActionContinue, nil
}

func (h *jsonHook) ProvisionOutput(id terraform.HookResourceIdentity, typeName string, msg string) {
	s := bufio.NewScanner(strings.NewReader(msg))
	s.Split(scanLines)
	for s.Scan() {
		line := strings.TrimRightFunc(s.Text(), unicode.IsSpace)
		if line != "" {
			h.view.Hook(json.NewProvisionProgress(id.Addr, typeName, line))
		}
	}
}

func (h *jsonHook) PreRefresh(id terraform.HookResourceIdentity, dk addrs.DeposedKey, priorState cty.Value) (terraform.HookAction, error) {
	idKey, idValue := format.ObjectValueID(priorState)
	h.view.Hook(json.NewRefreshStart(id.Addr, idKey, idValue))
	return terraform.HookActionContinue, nil
}

func (h *jsonHook) PostRefresh(id terraform.HookResourceIdentity, dk addrs.DeposedKey, priorState cty.Value, newState cty.Value) (terraform.HookAction, error) {
	idKey, idValue := format.ObjectValueID(newState)
	h.view.Hook(json.NewRefreshComplete(id.Addr, idKey, idValue))
	return terraform.HookActionContinue, nil
}
