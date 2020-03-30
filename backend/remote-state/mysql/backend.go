package mysql

import (
	"context"
	"database/sql"
	"fmt"

	// mysql import
	_ "github.com/go-sql-driver/mysql"
	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/schema"
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
				Description: "MySQL connection string; a `user:password@tcp(host:port)/dbname` URL",
			},

			"schema_name": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Name of the automatically managed MySQL database/schema to store state",
				Default:     "terraform_remote_state",
			},

			"skip_schema_creation": &schema.Schema{
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "If set to `true`, Terraform won't try to create the MySQL schema",
				Default:     false,
			},
		},
	}

	result := &Backend{Backend: s}
	result.Backend.ConfigureFunc = result.configure
	return result
}

//Backend type struct
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

	db, err := sql.Open("mysql", b.connStr)
	if err != nil {
		return err
	}

	// Prepare database, tables, & indexes.
	var query string

	if !data.Get("skip_schema_creation").(bool) {
		// list all database to see if it exists
		var count int
		query = `select count(1) from information_schema.schemata where lower(schema_name) = lower('%s')`
		if err := db.QueryRow(fmt.Sprintf(query, b.schemaName)).Scan(&count); err != nil {
			return err
		}

		// skip database (aka schema) creation if it already exists
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
	query = `CREATE TABLE IF NOT EXISTS %s.%s (
		id SERIAL NOT NULL PRIMARY KEY,
		name LONGTEXT,
		data LONGTEXT,
		UNIQUE INDEX %s (name(255))
	)`
	if _, err := db.Exec(fmt.Sprintf(query, b.schemaName, statesTableName, statesIndexName)); err != nil {
		return err
	}

	// Assign db after its schema is prepared.
	b.db = db

	return nil
}
