// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package providers

import (
	"sync"

	"github.com/hashicorp/terraform/internal/addrs"
)

// SchemaCache is a global cache of Schemas.
// This will be accessed by both core and the provider clients to ensure that
// large schemas are stored in a single location.
//
// FIXME: A global cache is inappropriate when Terraform Core is being
// used in a non-Terraform-CLI mode where we shouldn't assume that all
// calls share the same provider implementations. This would be better
// as a per-terraform.Context cache instead, or to have callers preload
// the schemas for the providers they intend to use and pass them in
// to terraform.NewContext so we don't need to load them at runtime.
var SchemaCache = &schemaCache{
	m: make(map[addrs.Provider]ProviderSchema),
}

// Global cache for provider schemas
// Cache the entire response to ensure we capture any new fields, like
// ServerCapabilities. This also serves to capture errors so that multiple
// concurrent calls resulting in an error can be handled in the same manner.
type schemaCache struct {
	mu sync.Mutex
	m  map[addrs.Provider]ProviderSchema
}

func (c *schemaCache) Set(p addrs.Provider, s ProviderSchema) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.m[p] = s
}

func (c *schemaCache) Get(p addrs.Provider) (ProviderSchema, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	s, ok := c.m[p]
	return s, ok
}
