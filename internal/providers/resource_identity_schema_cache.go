// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
)

// ResourceIdentitySchemasCache is a global cache of identity schemas.
// This will be accessed by both core and the provider clients to ensure that
// identity schemas are stored in a single location.
//
// FIXME: A global cache is inappropriate when Terraform Core is being
// used in a non-Terraform-CLI mode where we shouldn't assume that all
// calls share the same provider implementations. This would be better
// as a per-terraform.Context cache instead, or to have callers preload
// the schemas for the providers they intend to use and pass them in
// to terraform.NewContext so we don't need to load them at runtime.
var ResourceIdentitySchemasCache = &identitySchemasCache{
	m: make(map[addrs.Provider]ResourceIdentitySchemas),
}

// Global cache for resource identity schemas
type identitySchemasCache struct {
	mu sync.Mutex
	m  map[addrs.Provider]ResourceIdentitySchemas
}

func (c *identitySchemasCache) Set(p addrs.Provider, s ResourceIdentitySchemas) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.m[p] = s
}

func (c *identitySchemasCache) Get(p addrs.Provider) (ResourceIdentitySchemas, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	s, ok := c.m[p]
	return s, ok
}
