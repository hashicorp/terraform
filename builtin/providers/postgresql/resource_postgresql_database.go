package postgresql

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lib/pq"
)

func resourcePostgresqlDatabase() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgresqlDatabaseCreate,
		Read:   resourcePostgresqlDatabaseRead,
		Update: resourcePostgresqlDatabaseUpdate,
		Delete: resourcePostgresqlDatabaseDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"owner": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},
		},
	}
}

func resourcePostgresqlDatabaseCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	dbName := d.Get("name").(string)
	dbOwner := d.Get("owner").(string)
	connUsername := client.username

	var dbOwnerCfg string
	if dbOwner != "" {
		dbOwnerCfg = fmt.Sprintf("WITH OWNER=%s", pq.QuoteIdentifier(dbOwner))
	} else {
		dbOwnerCfg = ""
	}

	//needed in order to set the owner of the db if the connection user is not a superuser
	err = grantRoleMembership(conn, dbOwner, connUsername)
	if err != nil {
		return err
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
	connUsername := client.username
	dbOwner := d.Get("owner").(string)
	//needed in order to set the owner of the db if the connection user is not a superuser
	err = grantRoleMembership(conn, dbOwner, connUsername)
	if err != nil {
		return err
	}

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

func grantRoleMembership(conn *sql.DB, dbOwner string, connUsername string) error {
	if dbOwner != "" && dbOwner != connUsername {
		query := fmt.Sprintf("GRANT %s TO %s", pq.QuoteIdentifier(dbOwner), pq.QuoteIdentifier(connUsername))
		_, err := conn.Query(query)
		if err != nil {
			//is already member or role
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				return nil
			}
			return fmt.Errorf("Error granting membership: %s", err)
		}
	}
	return nil
}
