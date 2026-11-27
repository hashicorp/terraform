// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package policy

import (
	"github.com/hashicorp/terraform/internal/lang/ctyutil"
	"github.com/zclconf/go-cty/cty"
)

// ComparisonPaths collects the exact destinations compared for a resource type,
// regardless of which usage tree or endpoint mentions that type. Unused
// definitions do not contribute paths.
func (c *Config) ComparisonPaths(resourceType string) []cty.Path {
	if c == nil {
		return nil
	}
	var paths []cty.Path
	var collect func([]Relationship)
	collect = func(tree []Relationship) {
		for _, relationship := range tree {
			for _, match := range relationship.Matches {
				for _, key := range match {
					if relationship.KeyType == resourceType {
						paths = append(paths, ctyutil.NormalizePath(key.KeyPath))
					}
					if relationship.ValueType == resourceType {
						paths = append(paths, ctyutil.NormalizePath(key.ValuePath))
					}
				}
			}
			collect(relationship.Relationships)
		}
	}
	for _, tree := range c.Relationships {
		collect(tree)
	}
	return paths
}
