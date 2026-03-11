// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package pg

import (
	"database/sql"
	"fmt"

	"github.com/lib/pq"
	"github.com/zclconf/go-cty/cty"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

const (
	statesTableName = "states"
	statesIndexName = "states_by_name"
)

// New creates a new backend for Postgres remote state.
func New() backend.Backend {
	return &Backend{
		Base: backendbase.Base{
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"conn_str": {
						Type:        cty.String,
						Optional:    true,
						Description: "Postgres connection string; a `postgres://` URL",
					},
					"schema_name": {
						Type:        cty.String,
						Optional:    true,
						Description: "Name of the automatically managed Postgres schema to store state",
					},
					"skip_schema_creation": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "If set to `true`, Terraform won't try to create the Postgres schema",
					},
					"skip_table_creation": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "If set to `true`, Terraform won't try to create the Postgres table",
					},
					"skip_index_creation": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "If set to `true`, Terraform won't try to create the Postgres index",
					},
				},
			},
			SDKLikeDefaults: backendbase.SDKLikeDefaults{
				"conn_str": {
					EnvVars: []string{"PG_CONN_STR"},
				},
				"schema_name": {
					EnvVars:  []string{"PG_SCHEMA_NAME"},
					Fallback: "terraform_remote_state",
				},
				"skip_schema_creation": {
					EnvVars:  []string{"PG_SKIP_SCHEMA_CREATION"},
					Fallback: "false",
				},
				"skip_table_creation": {
					EnvVars:  []string{"PG_SKIP_TABLE_CREATION"},
					Fallback: "false",
				},
				"skip_index_creation": {
					EnvVars:  []string{"PG_SKIP_INDEX_CREATION"},
					Fallback: "false",
				},
			},
		},
	}
}

type Backend struct {
	backendbase.Base

	// The fields below are set from configure
	db         *sql.DB
	connStr    string
	schemaName string
}

func (b *Backend) Configure(configVal cty.Value) tfdiags.Diagnostics {
	data := backendbase.NewSDKLikeData(configVal)

	b.connStr = data.String("conn_str")
	b.schemaName = pq.QuoteIdentifier(data.String("schema_name"))

	db, err := sql.Open("postgres", b.connStr)
	if err != nil {
		return backendbase.ErrorAsDiagnostics(err)
	}

	// Prepare database schema, tables, & indexes.
	var query string

	if !data.Bool("skip_schema_creation") {
		// list all schemas to see if it exists
		var count int
		query = `select count(1) from information_schema.schemata where schema_name = $1`
		if err := db.QueryRow(query, data.String("schema_name")).Scan(&count); err != nil {
			return backendbase.ErrorAsDiagnostics(err)
		}

		// skip schema creation if schema already exists
		// `CREATE SCHEMA IF NOT EXISTS` is to be avoided if ever
		// a user hasn't been granted the `CREATE SCHEMA` privilege
		if count < 1 {
			// tries to create the schema
			query = `CREATE SCHEMA IF NOT EXISTS %s`
			if _, err := db.Exec(fmt.Sprintf(query, b.schemaName)); err != nil {
				return backendbase.ErrorAsDiagnostics(err)
			}
		}
	}

	if !data.Bool("skip_table_creation") {
		if _, err := db.Exec("CREATE SEQUENCE IF NOT EXISTS public.global_states_id_seq AS bigint"); err != nil {
			return backendbase.ErrorAsDiagnostics(err)
		}

		query = `CREATE TABLE IF NOT EXISTS %s.%s (
			id bigint NOT NULL DEFAULT nextval('public.global_states_id_seq') PRIMARY KEY,
			name text UNIQUE,
			data text
			)`
		if _, err := db.Exec(fmt.Sprintf(query, b.schemaName, statesTableName)); err != nil {
			return backendbase.ErrorAsDiagnostics(err)
		}
	}

	if !data.Bool("skip_index_creation") {
		query = `CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s.%s (name)`
		if _, err := db.Exec(fmt.Sprintf(query, statesIndexName, b.schemaName, statesTableName)); err != nil {
			return backendbase.ErrorAsDiagnostics(err)
		}
	}

	// Assign db after its schema is prepared.
	b.db = db

	return nil
}
