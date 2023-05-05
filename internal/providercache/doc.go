// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

// Package providercache contains the logic for auto-installing providers from
// packages obtained elsewhere, and for managing the local directories that
// serve as global or single-configuration caches of those auto-installed
// providers.
//
// It builds on the lower-level provider source functionality provided by
// the internal/getproviders package, adding the additional behaviors around
// obtaining the discovered providers and placing them in the cache
// directories for subsequent use.
package providercache
