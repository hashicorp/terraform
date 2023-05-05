// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package workdir models the various local artifacts and state we keep inside
// a Terraform "working directory".
//
// The working directory artifacts and settings are typically initialized or
// modified by "terraform init", after which they persist for use by other
// commands in the same directory, but are not visible to commands run in
// other working directories or on other computers.
//
// Although "terraform init" is the main command which modifies a workdir,
// other commands do sometimes make more focused modifications for settings
// which can typically change multiple times during a session, such as the
// currently-selected workspace name. Any command which modifies the working
// directory settings must discard and reload any objects which derived from
// those settings, because otherwise the existing objects will often continue
// to follow the settings that were present when they were created.
package workdir
