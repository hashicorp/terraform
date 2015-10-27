package postgresql

import (
	"fmt"
	"database/sql"

	"github.com/lib/pq"
	"github.com/hashicorp/terraform/helper/schema"
)


func resourcePostgresqlRoleCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*sql.DB)
	roleName := d.Get("name").(string)
	loginAttr := getLoginStr(d.Get("login").(bool))

	query := fmt.Sprintf("CREATE ROLE %s %s", pq.QuoteIdentifier(roleName), pq.QuoteIdentifier(loginAttr))
	_, err := conn.Query(query)
	if err != nil {
		return fmt.Errorf("Error creating role: %s", err)
	}

	d.SetId(roleName)

	return nil
}

func resourcePostgresqlRoleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*sql.DB)
	roleName := d.Get("name").(string)

	query := fmt.Sprintf("DROP ROLE %s", pq.QuoteIdentifier(roleName))
	_, err := conn.Query(query)
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourcePostgresqlRoleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*sql.DB)
	roleName := d.Get("name").(string)

	var canLogin bool
	err := conn.QueryRow("select rolcanlogin from pg_roles where rolname=$1", roleName).Scan(&canLogin)
	switch {
	case err == sql.ErrNoRows:
		d.SetId("")
		return nil
	case err != nil:
		return fmt.Errorf("Error reading info about role: %s", err)
	default:
		d.Set("login", canLogin)
		return nil
	}

	return nil
}

func resourcePostgresqlRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*sql.DB)
	roleName := d.Get("name").(string)

	if d.HasChange("login") {
		loginAttr := getLoginStr(d.Get("login").(bool))
		query := fmt.Sprintf("ALTER ROLE %s %s", pq.QuoteIdentifier(roleName), pq.QuoteIdentifier(loginAttr))
		_, err := conn.Query(query)
		if err != nil {
			return fmt.Errorf("Error updating login attribute for role: %s", err)
		}
	}

	return resourcePostgresqlRoleRead(d, meta)
}

func getLoginStr(canLogin bool) string {
	if canLogin {
		return "login"
	}
	return "nologin"
}
