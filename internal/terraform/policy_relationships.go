// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/depsfile"
	"github.com/hashicorp/terraform/internal/lang/ctyutil"
	"github.com/hashicorp/terraform/internal/lang/format"
	"github.com/hashicorp/terraform/internal/policy"
	"github.com/hashicorp/terraform/internal/states"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

// checkPolicyRelationships validates the relationships of the given policy
// client against the schemas of the providers in use. An invalid relationship
// fails the operation.
//
// TODO: Confirm with product whether an invalid relationship should fail the
// operation, or only skip policy evaluation and let the operation continue,
// as a policy setup failure does.
func (c *Context) checkPolicyRelationships(client policy.Client, config *configs.Config, state *states.State, locks map[addrs.Provider]*depsfile.ProviderLock) tfdiags.Diagnostics {
	if client == nil {
		return nil
	}
	policyConfig := client.Config()
	if policyConfig == nil || (len(policyConfig.Relationships) == 0 && len(policyConfig.Definitions) == 0) {
		return nil
	}

	schemas, diags := c.Schemas(config, state)
	if diags.HasErrors() {
		return diags
	}

	policyDiags := validatePolicyRelationships(policyConfig, schemas, locks)
	if !policyDiags.HasErrors() {
		return diags
	}

	for _, h := range c.hooks {
		h.PolicyDiagnostics(policyDiags)
	}

	return diags.Append(policyDiags.AsTerraformDiags())
}

// validatePolicyRelationships checks every key against the provider schemas,
// including definitions that no policy follows. Each path must select a string,
// number, or bool in its resource type. Resource types that no provider in use
// has are not checked.
func validatePolicyRelationships(config *policy.Config, schemas *Schemas, locks map[addrs.Provider]*depsfile.ProviderLock) policy.Diagnostics {
	if config == nil {
		return nil
	}
	var diags policy.Diagnostics

	providerAddrs := slices.SortedFunc(maps.Keys(schemas.Providers), func(a, b addrs.Provider) int {
		return strings.Compare(a.String(), b.String())
	})

	// Validate every distinct endpoint, even when relationship names repeat.
	seen := make(map[string][]cty.Path)
	validatePath := func(resourceType string, path cty.Path) {
		normalized := ctyutil.NormalizePath(path)
		for _, previous := range seen[resourceType] {
			if previous.Equals(normalized) {
				return
			}
		}
		seen[resourceType] = append(seen[resourceType], normalized)
		for _, providerAddr := range providerAddrs {
			schema, ok := schemas.Providers[providerAddr].ResourceTypes[resourceType]
			if !ok || schema.Body == nil {
				continue
			}
			typ, ok := schema.Body.TypeAtPath(path)
			if ok && typ.IsPrimitiveType() {
				continue
			}

			provider := providerAddr.String()
			if lock, ok := locks[providerAddr]; ok {
				provider = fmt.Sprintf("%s %s", provider, lock.Version())
			}
			name := strings.TrimPrefix(format.CtyPath(path), ".")
			detail := fmt.Sprintf("The resource type %s has no attribute %q in provider %s.", resourceType, name, provider)
			if ok {
				detail = fmt.Sprintf("The attribute %q of resource type %s in provider %s must have a primitive type (string, number, or bool), but has type %s.", name, resourceType, provider, typ.FriendlyName())
			}
			diags = append(diags, policy.NewErrorDiagnostic(
				"Invalid policy relationship",
				detail,
				policy.SetupErrorResult,
			))
		}
	}

	var validate func([]policy.Relationship)
	validate = func(relationships []policy.Relationship) {
		for _, relationship := range relationships {
			for _, match := range relationship.Matches {
				for _, key := range match {
					validatePath(relationship.KeyType, key.KeyPath)
					validatePath(relationship.ValueType, key.ValuePath)
				}
			}
			validate(relationship.Relationships)
		}
	}

	for _, name := range slices.Sorted(maps.Keys(config.Definitions)) {
		validate([]policy.Relationship{config.Definitions[name]})
	}
	for _, typ := range slices.Sorted(maps.Keys(config.Relationships)) {
		validate(config.Relationships[typ])
	}
	return diags
}
