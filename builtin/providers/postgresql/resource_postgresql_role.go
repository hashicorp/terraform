package postgresql

import (
	"database/sql"
	"fmt"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lib/pq"
)

func resourcePostgreSQLRole() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgreSQLRoleCreate,
		Read:   resourcePostgreSQLRoleRead,
		Update: resourcePostgreSQLRoleUpdate,
		Delete: resourcePostgreSQLRoleDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"login": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Default:  false,
			},
			"password": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: false,
				Default:  false,
			},
		},
	}
}

func resourcePostgreSQLRoleCreate(d *schema.ResourceData, meta interface{}) error {
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
		return errwrap.Wrapf("Error creating role: {{err}}", err)
	}

	d.SetId(roleName)

	return resourcePostgreSQLRoleRead(d, meta)
}

func resourcePostgreSQLRoleDelete(d *schema.ResourceData, meta interface{}) error {
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
		return errwrap.Wrapf("Error deleting role: {{err}}", err)
	}

	d.SetId("")

	return nil
}

func resourcePostgreSQLRoleRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	roleName := d.Get("name").(string)

	var canLogin bool
	err = conn.QueryRow("SELECT rolcanlogin FROM pg_roles WHERE rolname=$1", roleName).Scan(&canLogin)
	switch {
	case err == sql.ErrNoRows:
		d.SetId("")
		return nil
	case err != nil:
		return errwrap.Wrapf("Error reading role: {{err}}", err)
	default:
		d.Set("login", canLogin)
		return nil
	}
}

func resourcePostgreSQLRoleUpdate(d *schema.ResourceData, meta interface{}) error {
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
			return errwrap.Wrapf("Error updating login attribute for role: {{err}}", err)
		}

		d.SetPartial("login")
	}

	password := d.Get("password").(string)
	if d.HasChange("password") {
		encryptedCfg := getEncryptedStr(d.Get("encrypted").(bool))

		query := fmt.Sprintf("ALTER ROLE %s %s PASSWORD '%s'", pq.QuoteIdentifier(roleName), encryptedCfg, password)
		_, err := conn.Query(query)
		if err != nil {
			return errwrap.Wrapf("Error updating password attribute for role: {{err}}", err)
		}

		d.SetPartial("password")
	}

	if d.HasChange("encrypted") {
		encryptedCfg := getEncryptedStr(d.Get("encrypted").(bool))

		query := fmt.Sprintf("ALTER ROLE %s %s PASSWORD '%s'", pq.QuoteIdentifier(roleName), encryptedCfg, password)
		_, err := conn.Query(query)
		if err != nil {
			return errwrap.Wrapf("Error updating encrypted attribute for role: {{err}}", err)
		}

		d.SetPartial("encrypted")
	}

	d.Partial(false)
	return resourcePostgreSQLRoleRead(d, meta)
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
