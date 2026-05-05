// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/apparentlymart/go-versions/versions"
	version "github.com/hashicorp/go-version"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/zclconf/go-cty/cty"
)

// StateMigrationInstructions represents the sum of all state migration files within a
// configuration directory.
//
// A state migration file contains blocks that define how resource state has previously
// been stored for a given project. In combination with an updated Terraform configuration,
// the two pieces of information describe the source and destination of state that the user
// wishes to migrate.
//
// When creating a StateMigrationInstructions struct, calling code must ensure that there
// are no duplicated or mutually-exclusive pieces of information in the original file(s).
type StateMigrationInstructions struct {
	StateStoreProvider *RequiredProvider

	MigrateFromStateStore *StateStore
	MigrateFromBackend    *Backend
}

// StateMigrationFile represents a single state migration file within a configuration directory.
// A project can include multiple files of this type, and their contents is aggregated.
type StateMigrationFile struct {
	StateStoreProvider *RequiredProvider

	MigrateFromStateStore *StateStore
	MigrateFromBackend    *Backend
}

func loadStateMigrationFile(body hcl.Body) (*StateMigrationFile, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	file := &StateMigrationFile{}

	content, contentDiags := body.Content(stateMigrationFileSchema)
	diags = append(diags, contentDiags...)

	for _, block := range content.Blocks {
		switch block.Type {
		case "state_store_provider":
			p, pDiags := decodeStateStoreProviderBlock(block)
			diags = diags.Extend(pDiags)

			if file.StateStoreProvider != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Duplicate "state_store_provider" configuration block`,
					Detail:   `Only one "state_store_provider" block is allowed in a directory's .tfmigrate.hcl files.`,
					Subject:  block.DefRange.Ptr(),
				})
				continue // Keep file.StateStoreProvider as first parsed block in this scenario
			}

			if p != nil {
				file.StateStoreProvider = p
			}
		case "migrate_from_state_store":
			// TODO
		case "migrate_from_backend":
			b, bDiags := decodeMigrateFromBackendBlock(block)
			diags = diags.Extend(bDiags)

			if file.MigrateFromBackend != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Duplicate "migrate_from_backend" configuration block`,
					Detail:   `Only one "migrate_from_backend" block is allowed in a directory's .tfmigrate.hcl files.`,
					Subject:  block.DefRange.Ptr(),
				})
				continue // Keep file.MigrateFromBackend as first parsed block in this scenario
			}

			if b != nil {
				file.MigrateFromBackend = b
			}
		default:
			// We don't expect other block types in state migration files.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid block type",
				Detail:   fmt.Sprintf("This block type is not valid within a state migration file: %s", block.Type),
				Subject:  block.DefRange.Ptr(),
			})
		}
	}

	// Check for mutually exclusive blocks, etc.

	// Defining two conflicting sources of state for migration.
	if file.MigrateFromBackend != nil && file.MigrateFromStateStore != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid combination of "migrate_from_backend" and "migrate_from_state_store"`,
			Detail:   `The "migrate_from_backend" and "migrate_from_state_store" blocks are mutually-exclusive, only one should be used in a directory's .tfmigrate.hcl files..`,
		})
	}
	// Unnecessary state store-related data supplied alongside description of a backend.
	if file.MigrateFromBackend != nil && file.StateStoreProvider != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid combination of "migrate_from_backend" and "state_store_provider"`,
			Detail:   `The "state_store_provider" block can only be used in combination with "migrate_from_state_store" blocks. Either remove the unused "state_store_provider" block, or replace the "migrate_from_backend" block with a "migrate_from_state_store" block.`,
		})
	}

	return file, diags
}

func decodeStateStoreProviderBlock(block *hcl.Block) (*RequiredProvider, hcl.Diagnostics) {
	// state_store_provider blocks are similar to required_provider blocks but different, so we need logic
	// similar to that in decodeProviderRequirementsBlock but distinct. E.g. version constraints must be
	// exact versions, not a range. The similarity is sufficient that we can return a RequiredProvider pointer.

	var diags hcl.Diagnostics
	attrs, hclDiags := block.Body.JustAttributes()
	diags = diags.Extend(hclDiags)

	// Only one provider should be in the block
	localNames := slices.Collect(maps.Keys(attrs))
	if len(localNames) != 1 {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Unexpected number of providers described in "state_store_provider" configuration block.`,
			Detail:   fmt.Sprintf(`The "state_store_provider" block is only expected to include a single provider, but %d were found.`, len(localNames)),
			Subject:  block.DefRange.Ptr(),
		})
		return nil, diags
	}
	localName := localNames[0] // Local name
	attr := attrs[localName]   // Block containing source and version info

	// verify that the local name is already localized or produce an error.
	nameDiags := checkProviderNameNormalized(localName, attr.Expr.Range())
	if nameDiags.HasErrors() {
		diags = append(diags, nameDiags...)
		return nil, diags
	}

	kvs, mapDiags := hcl.ExprMap(attr.Expr)
	if mapDiags.HasErrors() {
		diags = append(diags, &hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid "state_store_provider" object`,
			Detail:   "The provider described inside state_store_provider must be an object",
			Subject:  attr.Expr.Range().Ptr(),
		})
		return nil, diags
	}

	// Process the data inside the object describing the provider
	ssProvider := RequiredProvider{
		Name:      localName,
		DeclRange: attr.Range,
	}
	for _, kv := range kvs {
		key, keyDiags := kv.Key.Value(nil)
		if keyDiags.HasErrors() {
			diags = append(diags, keyDiags...)
			return nil, diags
		}

		if key.Type() != cty.String {
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid Attribute",
				Detail:   fmt.Sprintf("Invalid attribute value for provider requirement described by state_store_provider block: %#v", key),
				Subject:  kv.Key.Range().Ptr(),
			})
			return nil, diags
		}

		switch key.AsString() {
		case "version":
			vc := VersionConstraint{
				DeclRange: attr.Range,
			}

			constraint, valDiags := kv.Value.Value(nil)
			if valDiags.HasErrors() || !constraint.Type().Equals(cty.String) {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid version constraint",
					Detail:   "Version must be specified as a string.",
					Subject:  kv.Value.Range().Ptr(),
				})
				continue
			}

			constraintStr := constraint.AsString()
			constraints, err := version.NewConstraint(constraintStr)
			if err != nil {
				// NewConstraint doesn't return user-friendly errors, so we'll just
				// ignore the provided error and produce our own generic one.
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid version constraint",
					Detail:   "This string does not use correct version constraint syntax.",
					Subject:  kv.Value.Range().Ptr(),
				})
				return nil, diags
			}

			// Assert we have a single constraint, for a specific version
			if len(constraints) != 1 {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid version constraint",
					Detail:   "The version attribute inside the state_store_provider block must specify a single, specific version (e.g. \"= 1.0.0\").",
					Subject:  kv.Value.Range().Ptr(),
				})
				return nil, diags
			}

			// A constraint to use v1.2.3 could have an = operator or no operator at all.
			constraintStr = strings.TrimPrefix(constraintStr, "=") // Remove a preceding `=`, if it exists.
			constraintStr = strings.TrimSpace(constraintStr)       // There might have been whitespace between the operator and the version.

			_, err = versions.ParseVersion(constraintStr)
			if err != nil {
				// Errors indicate that the constraint wasn't a specific version.
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Non-specific version constraint in "state_store_provider" configuration block`,
					Detail:   "The version constraint defined in a state_store_provider block must specify a single, specific version (e.g. \"= 1.0.0\", or \"1.0.0\").",
					Subject:  kv.Value.Range().Ptr(),
				})
				return nil, diags
			}

			// We capture the required version as a constraint, but
			// we know the constraint is to a single version.
			vc.Required = constraints
			ssProvider.Requirement = vc

		case "source":
			source, err := kv.Value.Value(nil)
			if err != nil || !source.Type().Equals(cty.String) {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Invalid source",
					Detail:   "Source must be specified as a string.",
					Subject:  kv.Value.Range().Ptr(),
				})
				return nil, diags
			}

			fqn, sourceDiags := addrs.ParseProviderSourceString(source.AsString())
			if sourceDiags.HasErrors() {
				hclDiags := sourceDiags.ToHCL()
				// The diagnostics from ParseProviderSourceString don't contain
				// source location information because it has no context to compute
				// them from, and so we'll add those in quickly here before we
				// return.
				for _, diag := range hclDiags {
					if diag.Subject == nil {
						diag.Subject = kv.Value.Range().Ptr()
					}
				}
				diags = append(diags, hclDiags...)
				return nil, diags
			}

			ssProvider.Source = source.AsString()
			ssProvider.Type = fqn
		default:
			diags = append(diags, &hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid state_store_provider object",
				Detail:   `state_store_provider objects can only contain "version" and "source" attributes.`,
				Subject:  kv.Key.Range().Ptr(),
			})
			return nil, diags
		}

	}

	return &ssProvider, diags
}

func decodeMigrateFromBackendBlock(block *hcl.Block) (*Backend, hcl.Diagnostics) {
	// migrate_from_backend blocks are essentially the same as backend blocks, so reuse logic.
	return decodeBackendBlock(block)
}

// stateMigrationFileSchema is the schema for a .tfmigrate.hcl file, for use with
// the `state migrate` command.
// Whereas the current Terraform config (.tf) defines the destination that state should
// be migrated to, these files define how a backend or state store was previously configured.
// Due to this, these files define the source where migrated state is copied from.
var stateMigrationFileSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type: "state_store_provider",
		},
		{
			Type:       "migrate_from_state_store",
			LabelNames: []string{"type"},
		},
		{
			Type:       "migrate_from_backend",
			LabelNames: []string{"type"},
		},
	},
}
