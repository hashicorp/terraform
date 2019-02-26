package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
	_ "github.com/lib/pq"
)

const (
	statesTableName = "states"
	statesIndexName = "states_by_name"
)

// New creates a new backend for Postgres remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"conn_str": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "Postgres connection string; a `postgres://` URL",
			},

			"schema_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the automatically managed Postgres schema to store state",
				Default:     "terraform_remote_backend",
			},
		},
	}

	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure
	return result
}

type Backend struct {
	*schema.Backend

	// The fields below are set from configure
	db         *sql.DB
	configData *schema.ResourceData
	connStr    string
	schemaName string
}

func (b *Backend) configure(ctx context.Context) error {
	// Grab the resource data
	b.configData = schema.FromContextBackendConfig(ctx)
	data := b.configData

	b.connStr = data.Get("conn_str").(string)
	b.schemaName = data.Get("schema_name").(string)

	db, err := sql.Open("postgres", b.connStr)
	if err != nil {
		return err
	}

	// Prepare database schema, tables, & indexes.
	var query string
	query = `CREATE SCHEMA IF NOT EXISTS %s`
	if _, err := db.Query(fmt.Sprintf(query, b.schemaName)); err != nil {
		return err
	}
	query = `CREATE TABLE IF NOT EXISTS %s.%s (
		id SERIAL PRIMARY KEY,
		name TEXT,
		data TEXT
	)`
	if _, err := db.Query(fmt.Sprintf(query, b.schemaName, statesTableName)); err != nil {
		return err
	}
	query = `CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s.%s (name)`
	if _, err := db.Query(fmt.Sprintf(query, statesIndexName, b.schemaName, statesTableName)); err != nil {
		return err
	}

	// Assign db after its schema is prepared.
	b.db = db

	return nil
}
