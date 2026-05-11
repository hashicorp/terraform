// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"
	"maps"
	"slices"

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

	// fromBlockSource is the source range of the 'from' block in the HCL file,
	// intended to be used in error diagnostics from parsing.
	fromBlockSource *hcl.Range
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
				file.fromBlockSource = &block.DefRange
			}
		case "from":
			if file.MigrateFromStateStore != nil || file.MigrateFromBackend != nil {
				// A from block has already been parsed.
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Duplicate "from" configuration block`,
					Detail:   `Only one "from" block is allowed in a directory's .tfmigrate.hcl files.`,
					Subject:  block.DefRange.Ptr(),
				})
				continue
			}

			// We're parsing the first encountered 'from' block.
			// There could still be duplications within that block, which is detected by the function.
			i, fromDiags := decodeFromBlock(block)
			diags = diags.Extend(fromDiags)

			if !fromDiags.HasErrors() {
				file.fromBlockSource = &block.DefRange

				// Only one of the below is non-nil
				file.MigrateFromStateStore = i.MigrateFromStateStore
				file.MigrateFromBackend = i.MigrateFromBackend
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
			Summary:  `Invalid combination of "backend" and "state_store"`,
			Detail:   `The "backend" and "state_store" blocks are mutually-exclusive inside a "from" block. Only one should be used in a directory's .tfmigrate.hcl files.`,
			Subject:  file.fromBlockSource, // We can blame the 'from' block as being invalid.
		})
	}
	// Unnecessary state store-related data supplied alongside description of a backend.
	if file.MigrateFromBackend != nil && file.StateStoreProvider != nil {
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  `Invalid combination of "backend" and "state_store_provider"`,
			Detail:   `The "state_store_provider" block can only be used in combination with a "state_store" block. Either remove the unused "state_store_provider" block, or update your "from" block to contain a "state_store" block instead.`,
			// No Subject because we don't know which is correct or incorrect.
		})
	}

	return file, diags
}

// decodeFromBlock decodes a 'from' block that can only contain one of 'state_store' or 'backend' blocks.
func decodeFromBlock(block *hcl.Block) (*StateMigrationInstructions, hcl.Diagnostics) {
	var diags hcl.Diagnostics
	fromData := StateMigrationInstructions{}

	fromContent, fromContentDiags := block.Body.Content(fromBlockSchema)
	diags = diags.Extend(fromContentDiags)

	for _, block := range fromContent.Blocks {
		switch block.Type {
		case "state_store":
			ss, ssDiags := decodeStateStoreBlock(block)
			diags = diags.Extend(ssDiags)

			if fromData.MigrateFromStateStore != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Duplicate "state_store" configuration block`,
					Detail:   `Only one "state_store" block, nested in a "from" block, is allowed in a directory's .tfmigrate.hcl files.`,
					Subject:  block.DefRange.Ptr(),
				})
				continue // Keep fromData.MigrateFromStateStore as first parsed block in this scenario
			}

			if ss != nil {
				fromData.MigrateFromStateStore = ss
			}
		case "backend":
			b, bDiags := decodeBackendBlock(block)
			diags = diags.Extend(bDiags)

			if fromData.MigrateFromBackend != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Duplicate "backend" configuration block`,
					Detail:   `Only one "backend" block, nested in a "from" block, is allowed in a directory's .tfmigrate.hcl files.`,
					Subject:  block.DefRange.Ptr(),
				})
				continue // Keep fromData.MigrateFromBackend as first parsed block in this scenario
			}

			if b != nil {
				fromData.MigrateFromBackend = b
			}
		default:
			// We don't expect other block types nested inside from blocks.
			diags = diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Invalid block type",
				Detail:   fmt.Sprintf("This block type is not valid to be nested inside 'from' blocks within a state migration file: %s", block.Type),
				Subject:  block.DefRange.Ptr(),
			})
		}
	}

	return &fromData, diags
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

			versionString, valDiags := kv.Value.Value(nil)
			if valDiags.HasErrors() || !versionString.Type().Equals(cty.String) {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Invalid provider version in "state_store_provider" configuration block`,
					Detail:   "Version must be a string, specifying a single version.",
					Subject:  kv.Value.Range().Ptr(),
				})
				continue
			}

			v, err := versions.ParseVersion(versionString.AsString())
			if err != nil {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Invalid provider version in "state_store_provider" configuration block`,
					Detail:   "The version attribute must specify a single, specific version (e.g. \"1.0.0\") and cannot be a version constraint with an operator.",
					Subject:  kv.Value.Range().Ptr(),
				})
				return nil, diags
			}

			// We ensure user input can be parsed as a version, but we need to
			// create a constraint to be part of the returned RequiredProvider struct.
			// The constraint will pin to a specific version set by the config.
			constraints, err := version.NewConstraint(v.String())
			if err != nil {
				// NewConstraint doesn't return user-friendly errors, so we'll just
				// ignore the provided error and produce our own generic one.
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Unable to create version constraint from provider version`,
					Detail:   fmt.Sprintf("Terraform was unable to create an 'exact' version constraint from the provided version string: %s.", v.String()),
					Subject:  kv.Value.Range().Ptr(),
				})
				return nil, diags
			}

			vc.Required = constraints
			ssProvider.Requirement = vc

		case "source":
			source, err := kv.Value.Value(nil)
			if err != nil || !source.Type().Equals(cty.String) {
				diags = append(diags, &hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  `Invalid source in "state_store_provider" configuration block`,
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
			Type: "from",
		},
	},
}

// fromBlockSchema is the schema for 'from' blocks within .tfmigrate.hcl files.
var fromBlockSchema = &hcl.BodySchema{
	Blocks: []hcl.BlockHeaderSchema{
		{
			Type:       "state_store",
			LabelNames: []string{"type"},
		},
		{
			Type:       "backend",
			LabelNames: []string{"type"},
		},
	},
}
