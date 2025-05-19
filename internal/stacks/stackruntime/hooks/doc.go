// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

// Package hooks is part of an optional API for callers to get realtime
// notifications of various events during the stack runtime's plan and apply
// processes.
//
// [stackruntime.Hooks] is the main entry-point into this API. This package
// contains supporting types and functions that hook implementers will typically
// need.
package hooks
