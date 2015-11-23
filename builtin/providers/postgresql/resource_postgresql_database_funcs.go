package postgresql

import (
	"database/sql"
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lib/pq"
)

func resourcePostgresqlDatabaseCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	dbName := d.Get("name").(string)
	dbOwner := d.Get("owner").(string)

	var dbOwnerCfg string
	if dbOwner != "" {
		dbOwnerCfg = fmt.Sprintf("WITH OWNER=%s", pq.QuoteIdentifier(dbOwner))
	} else {
		dbOwnerCfg = ""
	}

	query := fmt.Sprintf("CREATE DATABASE %s %s", pq.QuoteIdentifier(dbName), dbOwnerCfg)
	_, err = conn.Query(query)
	if err != nil {
		return fmt.Errorf("Error creating postgresql database %s: %s", dbName, err)
	}

	d.SetId(dbName)

	return nil
}

func resourcePostgresqlDatabaseDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	dbName := d.Get("name").(string)

	query := fmt.Sprintf("DROP DATABASE %s", pq.QuoteIdentifier(dbName))
	_, err = conn.Query(query)
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourcePostgresqlDatabaseRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	dbName := d.Get("name").(string)

	var owner string
	err = conn.QueryRow("SELECT pg_catalog.pg_get_userbyid(d.datdba) from pg_database d WHERE datname=$1", dbName).Scan(&owner)
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
}

func resourcePostgresqlDatabaseUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	dbName := d.Get("name").(string)

	if d.HasChange("owner") {
		owner := d.Get("owner").(string)
		if owner != "" {
			query := fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", pq.QuoteIdentifier(dbName), pq.QuoteIdentifier(owner))
			_, err := conn.Query(query)
			if err != nil {
				return fmt.Errorf("Error updating owner for database: %s", err)
			}
		}
	}

	return resourcePostgresqlDatabaseRead(d, meta)
}

