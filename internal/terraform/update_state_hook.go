// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

// updateStateHook calls the PostStateUpdate hook with the current state.
func updateStateHook(ctx EvalContext) error {
	// PostStateUpdate requires that the state be locked and safe to read for
	// the duration of the call.
	stateSync := ctx.State()
	state := stateSync.Lock()
	defer stateSync.Unlock()

	// Call the hook
	err := ctx.Hook(func(h Hook) (HookAction, error) {
		return h.PostStateUpdate(state)
	})
	return err
}
