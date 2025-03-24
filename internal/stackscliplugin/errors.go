// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stackscliplugin

import (
	"errors"
	"fmt"
)

var (
	// ErrStacksCLIPluginNotSupported is the error returned when the upstream HCP Terraform does not
	// have a manifest.
	ErrStacksCLIPluginNotSupported = errors.New("stacks-cli plugin is not supported by the remote version of Terraform Enterprise")

	// ErrRequestCanceled is the error returned when the context was cancelled.
	ErrRequestCanceled = errors.New("request was canceled")

	// ErrArchNotSupported is the error returned when the stacks-cli plugin does not have a build for the
	// current OS/Architecture.
	ErrArchNotSupported = errors.New("stacks-cli plugin is not supported by your computer architecture/operating system")

	// ErrStacksCLIPluginNotFound is the error returned when the stacks-cliplugin manifest points to a location
	// that was does not exist.
	ErrStacksCLIPluginNotFound = errors.New("stacks-cli plugin download was not found in the location specified in the manifest")
)

// ErrQueryFailed is the error returned when the stacks-cliplugin http client request fails
type ErrQueryFailed struct {
	inner error
}

// ErrStacksCLIPluginNotVerified is the error returned when the archive authentication process fails
type ErrStacksCLIPluginNotVerified struct {
	inner error
}

// Error returns a string representation of ErrQueryFailed
func (e ErrQueryFailed) Error() string {
	return fmt.Sprintf("failed to fetch stacks-cli plugin from HCP Terraform: %s", e.inner)
}

// Unwrap returns the inner error of ErrQueryFailed
func (e ErrQueryFailed) Unwrap() error {
	// Return the inner error.
	return e.inner
}

// Error returns the string representation of ErrStacksCLIPluginNotVerified
func (e ErrStacksCLIPluginNotVerified) Error() string {
	return fmt.Sprintf("failed to verify stacks-cli plugin. Ensure that the referenced plugin is the official HashiCorp distribution: %s", e.inner)
}

// Unwrap returns the inner error of ErrStacksCLIPluginNotVerified
func (e ErrStacksCLIPluginNotVerified) Unwrap() error {
	return e.inner
}
