package postgresql

import (
	"fmt"
	"database/sql"

	"github.com/lib/pq"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourcePostgresqlDbCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*sql.DB)
	dbName := d.Get("name").(string)
	dbOwner := d.Get("owner").(string)

	query := fmt.Sprintf("CREATE DATABASE %s WITH OWNER=%s", pq.QuoteIdentifier(dbName), pq.QuoteIdentifier(dbOwner))
	_, err := conn.Query(query)
	if err != nil {
		return fmt.Errorf("Error creating postgresql database: %s", err)
	}

	d.SetId(dbName)

	return nil
}

func resourcePostgresqlDbDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*sql.DB)
	dbName := d.Get("name").(string)

	query := fmt.Sprintf("DROP DATABASE %s", pq.QuoteIdentifier(dbName))
	_, err := conn.Query(query)
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourcePostgresqlDbRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*sql.DB)
	dbName := d.Get("name").(string)

	var owner string
	err := conn.QueryRow("SELECT pg_catalog.pg_get_userbyid(d.datdba) from pg_database d WHERE datname=$1", dbName).Scan(&owner)
	switch {
	case err == sql.ErrNoRows:
		d.SetId("")
		return nil
	case err != nil:
		return fmt.Errorf("Error reading info about database: %s", err)
	default:
		d.Set("owner", owner)
		return nil
	}

	return nil
}

func resourcePostgresqlDbUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*sql.DB)
	dbName := d.Get("name").(string)

	if d.HasChange("owner") {
		owner := d.Get("owner").(string)
		query := fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", pq.QuoteIdentifier(dbName), pq.QuoteIdentifier(owner))
		_, err := conn.Query(query)
		if err != nil {
			return fmt.Errorf("Error updating owner for database: %s", err)
		}
	}

	return resourcePostgresqlDbRead(d, meta)
}
