// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package deferring deals with the problem of keeping track of
// "deferred actions", which means the situation where the planning of some
// objects is currently impossible due to incomplete information and so
// Terraform explicitly defers dealing with them until the next plan/apply
// round, while still allowing the operator to apply the partial plan so
// that there will be more information available next time.
package deferring
