// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package tfdiags is a utility package for representing errors and
// warnings in a manner that allows us to produce good messages for the
// user.
//
// "diag" is short for "diagnostics", and is meant as a general word for
// feedback to a user about potential or actual problems.
//
// A design goal for this package is for it to be able to provide rich
// messaging where possible but to also be pragmatic about dealing with
// generic errors produced by system components that _can't_ provide
// such rich messaging. As a consequence, the main types in this package --
// Diagnostics and Diagnostic -- are designed so that they can be "smuggled"
// over an error channel and then be unpacked at the other end, so that
// error diagnostics (at least) can transit through APIs that are not
// aware of this package.
package tfdiags
