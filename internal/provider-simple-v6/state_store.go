// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package simple

// AttributeName can be overwritten using -ldflags at build time.
// This enables E2E tests using this provider implementation to build different
// versions of the same provider with different schemas.
var AttributeName string = "default"
