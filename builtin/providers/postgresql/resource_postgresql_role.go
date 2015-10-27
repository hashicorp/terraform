package postgresql

import (
	"database/sql"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lib/pq"
)

func resourcePostgresqlRole() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgresqlRoleCreate,
		Read:   resourcePostgresqlRoleRead,
		Update: resourcePostgresqlRoleUpdate,
		Delete: resourcePostgresqlRoleDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"login": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Default:  false,
			},
			"password": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"encrypted": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Default:  false,
			},
		},
	}
}

func resourcePostgresqlRoleCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	roleName := d.Get("name").(string)
	loginAttr := getLoginStr(d.Get("login").(bool))
	password := d.Get("password").(string)

	encryptedCfg := getEncryptedStr(d.Get("encrypted").(bool))

	query := fmt.Sprintf("CREATE ROLE %s %s %s PASSWORD '%s'", pq.QuoteIdentifier(roleName), loginAttr, encryptedCfg, password)
	_, err = conn.Query(query)
	if err != nil {
		return fmt.Errorf("Error creating role: %s", err)
	}

	d.SetId(roleName)

	return nil
}

func resourcePostgresqlRoleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	roleName := d.Get("name").(string)

	query := fmt.Sprintf("DROP ROLE %s", pq.QuoteIdentifier(roleName))
	_, err = conn.Query(query)
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func resourcePostgresqlRoleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	roleName := d.Get("name").(string)

	var canLogin bool
	err = conn.QueryRow("select rolcanlogin from pg_roles where rolname=$1", roleName).Scan(&canLogin)
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
}

func resourcePostgresqlRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	d.Partial(true)

	roleName := d.Get("name").(string)

	if d.HasChange("login") {
		loginAttr := getLoginStr(d.Get("login").(bool))
		query := fmt.Sprintf("ALTER ROLE %s %s", pq.QuoteIdentifier(roleName), pq.QuoteIdentifier(loginAttr))
		_, err := conn.Query(query)
		if err != nil {
			return fmt.Errorf("Error updating login attribute for role: %s", err)
		}

		d.SetPartial("login")
	}

	password := d.Get("password").(string)
	if d.HasChange("password") {
		encryptedCfg := getEncryptedStr(d.Get("encrypted").(bool))

		query := fmt.Sprintf("ALTER ROLE %s %s PASSWORD '%s'", pq.QuoteIdentifier(roleName), encryptedCfg, password)
		_, err := conn.Query(query)
		if err != nil {
			return fmt.Errorf("Error updating password attribute for role: %s", err)
		}

		d.SetPartial("password")
	}

	if d.HasChange("encrypted") {
		encryptedCfg := getEncryptedStr(d.Get("encrypted").(bool))

		query := fmt.Sprintf("ALTER ROLE %s %s PASSWORD '%s'", pq.QuoteIdentifier(roleName), encryptedCfg, password)
		_, err := conn.Query(query)
		if err != nil {
			return fmt.Errorf("Error updating encrypted attribute for role: %s", err)
		}

		d.SetPartial("encrypted")
	}

	d.Partial(false)
	return resourcePostgresqlRoleRead(d, meta)
}

func getLoginStr(canLogin bool) string {
	if canLogin {
		return "login"
	}
	return "nologin"
}

func getEncryptedStr(isEncrypted bool) string {
	if isEncrypted {
		return "encrypted"
	}
	return "unencrypted"
}
