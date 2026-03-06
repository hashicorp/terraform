// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

import "fmt"

// Registry holds all known migrations and provides lookup methods.
type Registry struct {
	migrations []Migration
}

// NewRegistry creates a new Registry populated with all built-in migrations.
func NewRegistry() *Registry {
	var all []Migration
	all = append(all, awsMigrations()...)
	all = append(all, azurermMigrations()...)
	all = append(all, terraformMigrations()...)

	return &Registry{
		migrations: all,
	}
}

// All returns every migration in the registry.
func (r *Registry) All() []Migration {
	return r.migrations
}

// Find returns the migration with the given ID, or an error if not found.
func (r *Registry) Find(id string) (Migration, error) {
	for _, m := range r.migrations {
		if m.ID() == id {
			return m, nil
		}
	}
	return Migration{}, fmt.Errorf("migration not found: %s", id)
}
