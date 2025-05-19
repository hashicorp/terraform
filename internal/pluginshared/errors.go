// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package pluginshared

import (
	"errors"
	"fmt"
)

var (
	// ErrPluginNotSupported is the error returned when the upstream HCP Terraform does not
	// have a manifest.
	ErrPluginNotSupported = errors.New("plugin is not supported by the remote version of Terraform Enterprise")

	// ErrRequestCanceled is the error returned when the context was cancelled.
	ErrRequestCanceled = errors.New("request was canceled")

	// ErrArchNotSupported is the error returned when the plugin does not have a build for the
	// current OS/Architecture.
	ErrArchNotSupported = errors.New("plugin is not supported by your computer architecture/operating system")

	// ErrPluginNotFound is the error returned when the plugin manifest points to a location
	// that was does not exist.
	ErrPluginNotFound = errors.New("plugin download was not found in the location specified in the manifest")
)

// ErrQueryFailed is the error returned when the plugin http client request fails
type ErrQueryFailed struct {
	inner error
}

// ErrCloudPluginNotVerified is the error returned when the archive authentication process fails
type ErrCloudPluginNotVerified struct {
	inner error
}

// Error returns a string representation of ErrQueryFailed
func (e ErrQueryFailed) Error() string {
	return fmt.Sprintf("failed to fetch plugin from HCP Terraform: %s", e.inner)
}

// Unwrap returns the inner error of ErrQueryFailed
func (e ErrQueryFailed) Unwrap() error {
	// Return the inner error.
	return e.inner
}

// Error returns the string representation of ErrCloudPluginNotVerified
func (e ErrCloudPluginNotVerified) Error() string {
	return fmt.Sprintf("failed to verify plugin. Ensure that the referenced plugin is the official HashiCorp distribution: %s", e.inner)
}

// Unwrap returns the inner error of ErrCloudPluginNotVerified
func (e ErrCloudPluginNotVerified) Unwrap() error {
	return e.inner
}
