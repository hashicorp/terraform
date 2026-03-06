// Copyright IBM Corp. 2014, 2026
// SPDX-License-Identifier: BUSL-1.1

package migrate

// Migration represents a named migration for a specific provider, consisting
// of one or more sub-migrations that are applied in order.
type Migration struct {
	Namespace     string         // e.g. "hashicorp"
	Provider      string         // e.g. "aws"
	Name          string         // e.g. "v3-to-v4"
	Description   string
	SubMigrations []SubMigration
}

// ID returns a unique identifier for this migration in the form
// "namespace/provider/name".
func (m Migration) ID() string {
	return m.Namespace + "/" + m.Provider + "/" + m.Name
}

// SubMigration represents a single transformation step within a migration.
type SubMigration struct {
	Name        string
	Description string
	Apply       func(filename string, src []byte) ([]byte, error)
}
