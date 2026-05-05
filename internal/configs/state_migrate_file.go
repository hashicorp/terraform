// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package configs

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
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
	StateStoreProvider *Provider

	MigrateFromStateStore *StateStore
	MigrateFromBackend    *Backend
}

// StateMigrationFile represents a single state migration file within a configuration directory.
// A project can include multiple files of this type, and their contents is aggregated.
type StateMigrationFile struct {
	StateStoreProvider *Provider

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
			// TODO
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
			Type:       "state_store_provider",
			LabelNames: []string{"type"},
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
