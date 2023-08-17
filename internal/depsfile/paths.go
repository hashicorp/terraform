// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package depsfile

// LockFilePath is the path, relative to a configuration's root module
// directory, where mnptu expects to find the dependency lock file for
// that configuration.
//
// This file is intended to be kept in version control, so it lives directly
// in the root module directory. The ".mnptu" prefix is intended to
// suggest that it's metadata about several types of objects that ultimately
// end up in the .mnptu directory after running "mnptu init".
const LockFilePath = ".mnptu.lock.hcl"

// DevOverrideFilePath is the path, relative to a configuration's root module
// directory, where mnptu will look to find a possible override file that
// represents a request to temporarily (within a single working directory only)
// use specific local directories in place of packages that would normally
// need to be installed from a remote location.
const DevOverrideFilePath = ".mnptu/dev-overrides.hcl"
