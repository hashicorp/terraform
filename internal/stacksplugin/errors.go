// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package stacksplugin

import (
	"errors"
	"fmt"
)

var (
	// ErrCloudPluginNotSupported is the error returned when the upstream HCP Terraform does not
	// have a manifest.
	ErrCloudPluginNotSupported = errors.New("cloud plugin is not supported by the remote version of Terraform Enterprise")

	// ErrRequestCanceled is the error returned when the context was cancelled.
	ErrRequestCanceled = errors.New("request was canceled")

	// ErrArchNotSupported is the error returned when the cloudplugin does not have a build for the
	// current OS/Architecture.
	ErrArchNotSupported = errors.New("cloud plugin is not supported by your computer architecture/operating system")

	// ErrCloudPluginNotFound is the error returned when the cloudplugin manifest points to a location
	// that was does not exist.
	ErrCloudPluginNotFound = errors.New("cloud plugin download was not found in the location specified in the manifest")
)

// ErrQueryFailed is the error returned when the cloudplugin http client request fails
type ErrQueryFailed struct {
	inner error
}

// ErrCloudPluginNotVerified is the error returned when the archive authentication process fails
type ErrStacksPluginNotVerified struct {
	inner error
}

// Error returns a string representation of ErrQueryFailed
func (e ErrQueryFailed) Error() string {
	return fmt.Sprintf("failed to fetch Stacks plugin from HCP Terraform: %s", e.inner)
}

// Unwrap returns the inner error of ErrQueryFailed
func (e ErrQueryFailed) Unwrap() error {
	// Return the inner error.
	return e.inner
}

// Error returns the string representation of ErrStacksPluginNotVerified
func (e ErrStacksPluginNotVerified) Error() string {
	return fmt.Sprintf("failed to verify Stacks plugin. Ensure that the referenced plugin is the official HashiCorp distribution: %s", e.inner)
}

// Unwrap returns the inner error of ErrStacksPluginNotVerified
func (e ErrStacksPluginNotVerified) Unwrap() error {
	return e.inner
}
