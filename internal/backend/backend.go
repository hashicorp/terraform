// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package backend provides interfaces that the CLI uses to interact with
// Terraform. A backend provides the abstraction that allows the same CLI
// to simultaneously support both local and remote operations for seamlessly
// using Terraform in a team environment.
package backend

import (
	"errors"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// DefaultStateName is the name of the default, initial state that every
// backend must have. This state cannot be deleted.
const DefaultStateName = "default"

var (
	// ErrDefaultWorkspaceNotSupported is returned when an operation does not
	// support using the default workspace, but requires a named workspace to
	// be selected.
	ErrDefaultWorkspaceNotSupported = errors.New("default workspace not supported\n" +
		"You can create a new workspace with the \"workspace new\" command.")

	// ErrWorkspacesNotSupported is an error returned when a caller attempts
	// to perform an operation on a workspace other than "default" for a
	// backend that doesn't support multiple workspaces.
	//
	// The caller can detect this to do special fallback behavior or produce
	// a specific, helpful error message.
	ErrWorkspacesNotSupported = errors.New("workspaces not supported")
)

// InitFn is used to initialize a new backend.
type InitFn func() Backend

// Backend is the minimal interface that must be implemented to enable Terraform.
type Backend interface {
	// ConfigSchema returns a description of the expected configuration
	// structure for the receiving backend.
	//
	// This method does not have any side-effects for the backend and can
	// be safely used before configuring.
	ConfigSchema() *configschema.Block

	// PrepareConfig checks the validity of the values in the given
	// configuration, and inserts any missing defaults, assuming that its
	// structure has already been validated per the schema returned by
	// ConfigSchema.
	//
	// This method does not have any side-effects for the backend and can
	// be safely used before configuring. It also does not consult any
	// external data such as environment variables, disk files, etc. Validation
	// that requires such external data should be deferred until the
	// Configure call.
	//
	// If error diagnostics are returned then the configuration is not valid
	// and must not subsequently be passed to the Configure method.
	//
	// This method may return configuration-contextual diagnostics such
	// as tfdiags.AttributeValue, and so the caller should provide the
	// necessary context via the diags.InConfigBody method before returning
	// diagnostics to the user.
	PrepareConfig(cty.Value) (cty.Value, tfdiags.Diagnostics)

	// Configure uses the provided configuration to set configuration fields
	// within the backend.
	//
	// The given configuration is assumed to have already been validated
	// against the schema returned by ConfigSchema and passed validation
	// via PrepareConfig.
	//
	// This method may be called only once per backend instance, and must be
	// called before all other methods except where otherwise stated.
	//
	// If error diagnostics are returned, the internal state of the instance
	// is undefined and no other methods may be called.
	Configure(cty.Value) tfdiags.Diagnostics

	// StateMgr returns the state manager for the given workspace name.
	//
	// If the returned state manager also implements statemgr.Locker then
	// it's the caller's responsibility to call Lock and Unlock as appropriate.
	//
	// If the named workspace doesn't exist, or if it has no state, it will
	// be created either immediately on this call or the first time
	// PersistState is called, depending on the state manager implementation.
	StateMgr(workspace string) (statemgr.Full, error)

	// DeleteWorkspace removes the workspace with the given name if it exists.
	//
	// DeleteWorkspace cannot prevent deleting a state that is in use. It is
	// the responsibility of the caller to hold a Lock for the state manager
	// belonging to this workspace before calling this method.
	DeleteWorkspace(name string, force bool) error

	// States returns a list of the names of all of the workspaces that exist
	// in this backend.
	Workspaces() ([]string, error)
}
