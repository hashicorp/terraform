package pg

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/legacy/helper/schema"
	"github.com/lib/pq"
	"github.com/lib/pq/auth/kerberos"
)

const (
	statesTableName = "states"
	statesIndexName = "states_by_name"
)

// New creates a new backend for Postgres remote state.
func New() backend.Backend {
	s := &schema.Backend{
		Schema: map[string]*schema.Schema{
			"conn_str": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Postgres connection string; a `postgres://` URL",
			},

			"schema_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the automatically managed Postgres schema to store state",
				Default:     "terraform_remote_state",
			},

			"skip_schema_creation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If set to `true`, Terraform won't try to create the Postgres schema",
				Default:     false,
			},

			"skip_table_creation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If set to `true`, Terraform won't try to create the Postgres table",
			},

			"skip_index_creation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If set to `true`, Terraform won't try to create the Postgres index",
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
	b.schemaName = pq.QuoteIdentifier(data.Get("schema_name").(string))
	pq.RegisterGSSProvider(func() (pq.GSS, error) { return kerberos.NewGSS() })

	db, err := sql.Open("postgres", b.connStr)
	if err != nil {
		return err
	}

	// Prepare database schema, tables, & indexes.
	var query string

	if !data.Get("skip_schema_creation").(bool) {
		// list all schemas to see if it exists
		var count int
		query = `select count(1) from information_schema.schemata where schema_name = $1`
		if err := db.QueryRow(query, data.Get("schema_name").(string)).Scan(&count); err != nil {
			return err
		}

		// skip schema creation if schema already exists
		// `CREATE SCHEMA IF NOT EXISTS` is to be avoided if ever
		// a user hasn't been granted the `CREATE SCHEMA` privilege
		if count < 1 {
			// tries to create the schema
			query = `CREATE SCHEMA IF NOT EXISTS %s`
			if _, err := db.Exec(fmt.Sprintf(query, b.schemaName)); err != nil {
				return err
			}
		}
	}

	if !data.Get("skip_table_creation").(bool) {
		if _, err := db.Exec("CREATE SEQUENCE IF NOT EXISTS public.global_states_id_seq AS bigint"); err != nil {
			return err
		}

		query = `CREATE TABLE IF NOT EXISTS %s.%s (
			id bigint NOT NULL DEFAULT nextval('public.global_states_id_seq') PRIMARY KEY,
			name text UNIQUE,
			data text
			)`
		if _, err := db.Exec(fmt.Sprintf(query, b.schemaName, statesTableName)); err != nil {
			return err
		}
	}

	if !data.Get("skip_index_creation").(bool) {
		query = `CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s.%s (name)`
		if _, err := db.Exec(fmt.Sprintf(query, statesIndexName, b.schemaName, statesTableName)); err != nil {
			return err
		}
	}

	// Assign db after its schema is prepared.
	b.db = db

	return nil
}
