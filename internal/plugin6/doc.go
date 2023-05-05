// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package plugin6

// plugin6 builds on types in package plugin to include support for plugin
// protocol v6. The main gRPC functions use by terraform (and initialized in
// init.go), such as Serve, are in the plugin package. The version of those
// functions in this package are used by various mocks and in tests.

// When provider protocol v5 is deprecated, some functions may need to be moved
// here, or the existing functions updated, before removing the plugin pacakge.
