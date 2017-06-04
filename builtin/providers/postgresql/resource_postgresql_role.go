package postgresql

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lib/pq"
)

const (
	roleBypassRLSAttr         = "bypass_row_level_security"
	roleConnLimitAttr         = "connection_limit"
	roleCreateDBAttr          = "create_database"
	roleCreateRoleAttr        = "create_role"
	roleEncryptedPassAttr     = "encrypted_password"
	roleInheritAttr           = "inherit"
	roleLoginAttr             = "login"
	roleNameAttr              = "name"
	rolePasswordAttr          = "password"
	roleReplicationAttr       = "replication"
	roleSkipDropRoleAttr      = "skip_drop_role"
	roleSkipReassignOwnedAttr = "skip_reassign_owned"
	roleSuperuserAttr         = "superuser"
	roleValidUntilAttr        = "valid_until"

	// Deprecated options
	roleDepEncryptedAttr = "encrypted"
)

func resourcePostgreSQLRole() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgreSQLRoleCreate,
		Read:   resourcePostgreSQLRoleRead,
		Update: resourcePostgreSQLRoleUpdate,
		Delete: resourcePostgreSQLRoleDelete,
		Exists: resourcePostgreSQLRoleExists,
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
				Default:     "infinity",
				Description: "Sets a date and time after which the role's password is no longer valid",
			},
			roleConnLimitAttr: {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      -1,
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
				Default:     true,
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
			roleSkipDropRoleAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Skip actually running the DROP ROLE command when removing a ROLE from PostgreSQL",
			},
			roleSkipReassignOwnedAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Skip actually running the REASSIGN OWNED command when removing a role from PostgreSQL",
			},
		},
	}
}

func resourcePostgreSQLRoleCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	c.catalogLock.Lock()
	defer c.catalogLock.Unlock()

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
					createOpts = append(createOpts, fmt.Sprintf("%s '%s'", opt.sqlKey, pqQuoteLiteral(val)))
				}
			case opt.hclKey == roleValidUntilAttr:
				switch {
				case v.(string) == "", strings.ToLower(v.(string)) == "infinity":
					createOpts = append(createOpts, fmt.Sprintf("%s '%s'", opt.sqlKey, "infinity"))
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
		// FIXME(seanc@): Work around ParAccel/AWS RedShift's ancient fork of PostgreSQL
		// createStr = " WITH " + createStr
		createStr = " " + createStr
	}

	query := fmt.Sprintf("CREATE ROLE %s%s", pq.QuoteIdentifier(roleName), createStr)
	_, err = conn.Query(query)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Error creating role %s: {{err}}", roleName), err)
	}

	d.SetId(roleName)

	return resourcePostgreSQLRoleReadImpl(d, meta)
}

func resourcePostgreSQLRoleDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	c.catalogLock.Lock()
	defer c.catalogLock.Unlock()

	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	txn, err := conn.Begin()
	if err != nil {
		return err
	}
	defer txn.Rollback()

	roleName := d.Get(roleNameAttr).(string)

	queries := make([]string, 0, 3)
	if !d.Get(roleSkipReassignOwnedAttr).(bool) {
		queries = append(queries, fmt.Sprintf("REASSIGN OWNED BY %s TO CURRENT_USER", pq.QuoteIdentifier(roleName)))
		queries = append(queries, fmt.Sprintf("DROP OWNED BY %s", pq.QuoteIdentifier(roleName)))
	}

	if !d.Get(roleSkipDropRoleAttr).(bool) {
		queries = append(queries, fmt.Sprintf("DROP ROLE %s", pq.QuoteIdentifier(roleName)))
	}

	if len(queries) > 0 {
		for _, query := range queries {
			_, err = conn.Query(query)
			if err != nil {
				return errwrap.Wrapf("Error deleting role: {{err}}", err)
			}
		}

		if err := txn.Commit(); err != nil {
			return errwrap.Wrapf("Error committing schema: {{err}}", err)
		}
	}

	d.SetId("")

	return nil
}

func resourcePostgreSQLRoleExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	c := meta.(*Client)
	c.catalogLock.RLock()
	defer c.catalogLock.RUnlock()

	conn, err := c.Connect()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	var roleName string
	err = conn.QueryRow("SELECT rolname FROM pg_catalog.pg_roles WHERE rolname=$1", d.Id()).Scan(&roleName)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, err
	}

	return true, nil
}

func resourcePostgreSQLRoleRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	c.catalogLock.RLock()
	defer c.catalogLock.RUnlock()

	return resourcePostgreSQLRoleReadImpl(d, meta)
}

func resourcePostgreSQLRoleReadImpl(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	roleId := d.Id()
	var roleSuperuser, roleInherit, roleCreateRole, roleCreateDB, roleCanLogin, roleReplication, roleBypassRLS bool
	var roleConnLimit int
	var roleName, roleValidUntil string
	err = conn.QueryRow("SELECT rolname, rolsuper, rolinherit, rolcreaterole, rolcreatedb, rolcanlogin, rolreplication, rolconnlimit, COALESCE(rolvaliduntil::TEXT, 'infinity'), rolbypassrls FROM pg_catalog.pg_roles WHERE rolname=$1", roleId).Scan(&roleName, &roleSuperuser, &roleInherit, &roleCreateRole, &roleCreateDB, &roleCanLogin, &roleReplication, &roleConnLimit, &roleValidUntil, &roleBypassRLS)
	switch {
	case err == sql.ErrNoRows:
		log.Printf("[WARN] PostgreSQL role (%s) not found", roleId)
		d.SetId("")
		return nil
	case err != nil:
		return errwrap.Wrapf("Error reading role: {{err}}", err)
	default:
		d.Set(roleNameAttr, roleName)
		d.Set(roleBypassRLSAttr, roleBypassRLS)
		d.Set(roleConnLimitAttr, roleConnLimit)
		d.Set(roleCreateDBAttr, roleCreateDB)
		d.Set(roleCreateRoleAttr, roleCreateRole)
		d.Set(roleEncryptedPassAttr, true)
		d.Set(roleInheritAttr, roleInherit)
		d.Set(roleLoginAttr, roleCanLogin)
		d.Set(roleReplicationAttr, roleReplication)
		d.Set(roleSkipDropRoleAttr, d.Get(roleSkipDropRoleAttr).(bool))
		d.Set(roleSkipReassignOwnedAttr, d.Get(roleSkipReassignOwnedAttr).(bool))
		d.Set(roleSuperuserAttr, roleSuperuser)
		d.Set(roleValidUntilAttr, roleValidUntil)
		d.SetId(roleName)
	}

	if !roleSuperuser {
		// Return early if not superuser user
		return nil
	}

	var rolePassword string
	err = conn.QueryRow("SELECT COALESCE(passwd, '') FROM pg_catalog.pg_shadow AS s WHERE s.usename = $1", roleId).Scan(&rolePassword)
	switch {
	case err == sql.ErrNoRows:
		return errwrap.Wrapf(fmt.Sprintf("PostgreSQL role (%s) not found in shadow database: {{err}}", roleId), err)
	case err != nil:
		return errwrap.Wrapf("Error reading role: {{err}}", err)
	default:
		d.Set(rolePasswordAttr, rolePassword)
		return nil
	}
}

func resourcePostgreSQLRoleUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	c.catalogLock.Lock()
	defer c.catalogLock.Unlock()

	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := setRoleName(conn, d); err != nil {
		return err
	}

	if err := setRoleBypassRLS(conn, d); err != nil {
		return err
	}

	if err := setRoleConnLimit(conn, d); err != nil {
		return err
	}

	if err := setRoleCreateDB(conn, d); err != nil {
		return err
	}

	if err := setRoleCreateRole(conn, d); err != nil {
		return err
	}

	if err := setRoleInherit(conn, d); err != nil {
		return err
	}

	if err := setRoleLogin(conn, d); err != nil {
		return err
	}

	if err := setRoleReplication(conn, d); err != nil {
		return err
	}

	if err := setRoleSuperuser(conn, d); err != nil {
		return err
	}

	if err := setRoleValidUntil(conn, d); err != nil {
		return err
	}

	return resourcePostgreSQLRoleReadImpl(d, meta)
}

func setRoleName(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(roleNameAttr) {
		return nil
	}

	oraw, nraw := d.GetChange(roleNameAttr)
	o := oraw.(string)
	n := nraw.(string)
	if n == "" {
		return errors.New("Error setting role name to an empty string")
	}

	query := fmt.Sprintf("ALTER ROLE %s RENAME TO %s", pq.QuoteIdentifier(o), pq.QuoteIdentifier(n))
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating role NAME: {{err}}", err)
	}
	d.SetId(n)

	return nil
}

func setRoleBypassRLS(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(roleBypassRLSAttr) {
		return nil
	}

	bypassRLS := d.Get(roleBypassRLSAttr).(bool)
	tok := "NOBYPASSRLS"
	if bypassRLS {
		tok = "BYPASSRLS"
	}
	roleName := d.Get(roleNameAttr).(string)
	query := fmt.Sprintf("ALTER ROLE %s WITH %s", pq.QuoteIdentifier(roleName), tok)
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating role BYPASSRLS: {{err}}", err)
	}

	return nil
}

func setRoleConnLimit(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(roleConnLimitAttr) {
		return nil
	}

	connLimit := d.Get(roleConnLimitAttr).(int)
	roleName := d.Get(roleNameAttr).(string)
	query := fmt.Sprintf("ALTER ROLE %s CONNECTION LIMIT %d", pq.QuoteIdentifier(roleName), connLimit)
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating role CONNECTION LIMIT: {{err}}", err)
	}

	return nil
}

func setRoleCreateDB(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(roleCreateDBAttr) {
		return nil
	}

	createDB := d.Get(roleCreateDBAttr).(bool)
	tok := "NOCREATEDB"
	if createDB {
		tok = "CREATEDB"
	}
	roleName := d.Get(roleNameAttr).(string)
	query := fmt.Sprintf("ALTER ROLE %s WITH %s", pq.QuoteIdentifier(roleName), tok)
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating role CREATEDB: {{err}}", err)
	}

	return nil
}

func setRoleCreateRole(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(roleCreateRoleAttr) {
		return nil
	}

	createRole := d.Get(roleCreateRoleAttr).(bool)
	tok := "NOCREATEROLE"
	if createRole {
		tok = "CREATEROLE"
	}
	roleName := d.Get(roleNameAttr).(string)
	query := fmt.Sprintf("ALTER ROLE %s WITH %s", pq.QuoteIdentifier(roleName), tok)
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating role CREATEROLE: {{err}}", err)
	}

	return nil
}

func setRoleInherit(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(roleInheritAttr) {
		return nil
	}

	inherit := d.Get(roleInheritAttr).(bool)
	tok := "NOINHERIT"
	if inherit {
		tok = "INHERIT"
	}
	roleName := d.Get(roleNameAttr).(string)
	query := fmt.Sprintf("ALTER ROLE %s WITH %s", pq.QuoteIdentifier(roleName), tok)
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating role INHERIT: {{err}}", err)
	}

	return nil
}

func setRoleLogin(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(roleLoginAttr) {
		return nil
	}

	login := d.Get(roleLoginAttr).(bool)
	tok := "NOLOGIN"
	if login {
		tok = "LOGIN"
	}
	roleName := d.Get(roleNameAttr).(string)
	query := fmt.Sprintf("ALTER ROLE %s WITH %s", pq.QuoteIdentifier(roleName), tok)
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating role LOGIN: {{err}}", err)
	}

	return nil
}

func setRoleReplication(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(roleReplicationAttr) {
		return nil
	}

	replication := d.Get(roleReplicationAttr).(bool)
	tok := "NOREPLICATION"
	if replication {
		tok = "REPLICATION"
	}
	roleName := d.Get(roleNameAttr).(string)
	query := fmt.Sprintf("ALTER ROLE %s WITH %s", pq.QuoteIdentifier(roleName), tok)
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating role REPLICATION: {{err}}", err)
	}

	return nil
}

func setRoleSuperuser(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(roleSuperuserAttr) {
		return nil
	}

	superuser := d.Get(roleSuperuserAttr).(bool)
	tok := "NOSUPERUSER"
	if superuser {
		tok = "SUPERUSER"
	}
	roleName := d.Get(roleNameAttr).(string)
	query := fmt.Sprintf("ALTER ROLE %s WITH %s", pq.QuoteIdentifier(roleName), tok)
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating role SUPERUSER: {{err}}", err)
	}

	return nil
}

func setRoleValidUntil(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(roleValidUntilAttr) {
		return nil
	}

	validUntil := d.Get(roleValidUntilAttr).(string)
	if validUntil == "" {
		return nil
	} else if strings.ToLower(validUntil) == "infinity" {
		validUntil = "infinity"
	}

	roleName := d.Get(roleNameAttr).(string)
	query := fmt.Sprintf("ALTER ROLE %s VALID UNTIL '%s'", pq.QuoteIdentifier(roleName), pqQuoteLiteral(validUntil))

	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating role VALID UNTIL: {{err}}", err)
	}

	return nil
}
