package postgresql

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lib/pq"
)

const (
	roleBypassRLSAttr     = "bypass_row_level_security"
	roleConnLimitAttr     = "connection_limit"
	roleCreateDBAttr      = "create_database"
	roleCreateRoleAttr    = "create_role"
	roleEncryptedPassAttr = "encrypted_password"
	roleInheritAttr       = "inherit"
	roleLoginAttr         = "login"
	roleNameAttr          = "name"
	rolePasswordAttr      = "password"
	roleReplicationAttr   = "replication"
	roleSuperuserAttr     = "superuser"
	roleValidUntilAttr    = "valid_until"

	// Deprecated options
	roleDepEncryptedAttr = "encrypted"
)

func resourcePostgreSQLRole() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgreSQLRoleCreate,
		Read:   resourcePostgreSQLRoleRead,
		Update: resourcePostgreSQLRoleUpdate,
		Delete: resourcePostgreSQLRoleDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			roleNameAttr: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the role",
			},
			rolePasswordAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Sensitive:   true,
				DefaultFunc: schema.EnvDefaultFunc("PGPASSWORD", nil),
				Description: "Sets the role's password",
			},
			roleDepEncryptedAttr: {
				Type:       schema.TypeString,
				Optional:   true,
				Deprecated: fmt.Sprintf("Rename PostgreSQL role resource attribute %q to %q", roleDepEncryptedAttr, roleEncryptedPassAttr),
			},
			roleEncryptedPassAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Control whether the password is stored encrypted in the system catalogs",
			},

			roleValidUntilAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Sets a date and time after which the role's password is no longer valid",
			},
			roleConnLimitAttr: {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "How many concurrent connections can be made with this role",
				ValidateFunc: validateConnLimit,
			},
			roleSuperuserAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: `Determine whether the new role is a "superuser"`,
			},
			roleCreateDBAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Define a role's ability to create databases",
			},
			roleCreateRoleAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Determine whether this role will be permitted to create new roles",
			},
			roleInheritAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: `Determine whether a role "inherits" the privileges of roles it is a member of`,
			},
			roleLoginAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Determine whether a role is allowed to log in",
			},
			roleReplicationAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Determine whether a role is allowed to initiate streaming replication or put the system in and out of backup mode",
			},
			roleBypassRLSAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Determine whether a role bypasses every row-level security (RLS) policy",
			},
		},
	}
}

func resourcePostgreSQLRoleCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return errwrap.Wrapf("Error connecting to PostgreSQL: {{err}}", err)
	}
	defer conn.Close()

	stringOpts := []struct {
		hclKey string
		sqlKey string
	}{
		{rolePasswordAttr, "PASSWORD"},
		{roleValidUntilAttr, "VALID UNTIL"},
	}
	intOpts := []struct {
		hclKey string
		sqlKey string
	}{
		{roleConnLimitAttr, "CONNECTION LIMIT"},
	}
	boolOpts := []struct {
		hclKey        string
		sqlKeyEnable  string
		sqlKeyDisable string
	}{
		{roleSuperuserAttr, "CREATEDB", "NOCREATEDB"},
		{roleCreateRoleAttr, "CREATEROLE", "NOCREATEROLE"},
		{roleInheritAttr, "INHERIT", "NOINHERIT"},
		{roleLoginAttr, "LOGIN", "NOLOGIN"},
		{roleReplicationAttr, "REPLICATION", "NOREPLICATION"},
		{roleBypassRLSAttr, "BYPASSRLS", "NOBYPASSRLS"},

		// roleEncryptedPassAttr is used only when rolePasswordAttr is set.
		// {roleEncryptedPassAttr, "ENCRYPTED", "UNENCRYPTED"},
	}

	createOpts := make([]string, 0, len(stringOpts)+len(intOpts)+len(boolOpts))

	for _, opt := range stringOpts {
		v, ok := d.GetOk(opt.hclKey)
		if !ok {
			continue
		}

		val := v.(string)
		if val != "" {
			switch {
			case opt.hclKey == rolePasswordAttr:
				if strings.ToUpper(v.(string)) == "NULL" {
					createOpts = append(createOpts, "PASSWORD NULL")
				} else {
					if d.Get(roleEncryptedPassAttr).(bool) {
						createOpts = append(createOpts, "ENCRYPTED")
					} else {
						createOpts = append(createOpts, "UNENCRYPTED")
					}
					escapedPassword := strconv.Quote(val)
					escapedPassword = strings.TrimLeft(escapedPassword, `"`)
					escapedPassword = strings.TrimRight(escapedPassword, `"`)
					createOpts = append(createOpts, fmt.Sprintf("%s '%s'", opt.sqlKey, escapedPassword))
				}
			case opt.hclKey == roleValidUntilAttr:
				switch {
				case v.(string) == "", strings.ToUpper(v.(string)) == "NULL":
					createOpts = append(createOpts, fmt.Sprintf("%s %s", opt.sqlKey, "'infinity'"))
				default:
					createOpts = append(createOpts, fmt.Sprintf("%s %s", opt.sqlKey, pq.QuoteIdentifier(val)))
				}
			default:
				createOpts = append(createOpts, fmt.Sprintf("%s %s", opt.sqlKey, pq.QuoteIdentifier(val)))
			}
		}
	}

	for _, opt := range intOpts {
		val := d.Get(opt.hclKey).(int)
		createOpts = append(createOpts, fmt.Sprintf("%s %d", opt.sqlKey, val))
	}

	for _, opt := range boolOpts {
		if opt.hclKey == roleEncryptedPassAttr {
			// This attribute is handled above in the stringOpts
			// loop.
			continue
		}
		val := d.Get(opt.hclKey).(bool)

		valStr := opt.sqlKeyDisable
		if val {
			valStr = opt.sqlKeyEnable
		}
		createOpts = append(createOpts, valStr)
	}

	roleName := d.Get(roleNameAttr).(string)
	createStr := strings.Join(createOpts, " ")
	if len(createOpts) > 0 {
		createStr = " WITH " + createStr
	}

	query := fmt.Sprintf("CREATE ROLE %s%s", pq.QuoteIdentifier(roleName), createStr)
	_, err = conn.Query(query)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Error creating role %s: {{err}}", roleName), err)
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

	roleName := d.Get(roleNameAttr).(string)

	query := fmt.Sprintf("DROP ROLE %s", pq.QuoteIdentifier(roleName))
	_, err = conn.Query(query)
	if err != nil {
		return errwrap.Wrapf("Error deleting role: {{err}}", err)
	}

	d.SetId("")

	return nil
}

func resourcePostgreSQLRoleRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	roleName := d.Get(roleNameAttr).(string)
	if roleName == "" {
		roleName = d.Id()
	}

	var roleCanLogin bool
	err = conn.QueryRow("SELECT rolcanlogin FROM pg_roles WHERE rolname=$1", roleName).Scan(&roleCanLogin)
	switch {
	case err == sql.ErrNoRows:
		log.Printf("[WARN] PostgreSQL database (%s) not found", d.Id())
		d.SetId("")
		return nil
	case err != nil:
		return errwrap.Wrapf("Error reading role: {{err}}", err)
	default:
		d.Set(roleNameAttr, roleName)
		d.Set(roleLoginAttr, roleCanLogin)
		d.Set("encrypted", true)
		d.SetId(roleName)
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

	roleName := d.Get(roleNameAttr).(string)

	if d.HasChange(roleLoginAttr) {
		loginAttr := getLoginStr(d.Get(roleLoginAttr).(bool))
		query := fmt.Sprintf("ALTER ROLE %s %s", pq.QuoteIdentifier(roleName), pq.QuoteIdentifier(loginAttr))
		_, err := conn.Query(query)
		if err != nil {
			return errwrap.Wrapf("Error updating login attribute for role: {{err}}", err)
		}

		d.SetPartial(roleLoginAttr)
	}

	password := d.Get(rolePasswordAttr).(string)
	if d.HasChange(rolePasswordAttr) {
		encryptedCfg := getEncryptedStr(d.Get("encrypted").(bool))

		query := fmt.Sprintf("ALTER ROLE %s %s PASSWORD '%s'", pq.QuoteIdentifier(roleName), encryptedCfg, password)
		_, err := conn.Query(query)
		if err != nil {
			return errwrap.Wrapf("Error updating password attribute for role: {{err}}", err)
		}

		d.SetPartial(rolePasswordAttr)
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
