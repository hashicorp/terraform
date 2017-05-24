package postgresql

import (
	"bytes"
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
	dbAllowConnsAttr = "allow_connections"
	dbCTypeAttr      = "lc_ctype"
	dbCollationAttr  = "lc_collate"
	dbConnLimitAttr  = "connection_limit"
	dbEncodingAttr   = "encoding"
	dbIsTemplateAttr = "is_template"
	dbNameAttr       = "name"
	dbOwnerAttr      = "owner"
	dbTablespaceAttr = "tablespace_name"
	dbTemplateAttr   = "template"
)

func resourcePostgreSQLDatabase() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgreSQLDatabaseCreate,
		Read:   resourcePostgreSQLDatabaseRead,
		Update: resourcePostgreSQLDatabaseUpdate,
		Delete: resourcePostgreSQLDatabaseDelete,
		Exists: resourcePostgreSQLDatabaseExists,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			dbNameAttr: {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The PostgreSQL database name to connect to",
			},
			dbOwnerAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The ROLE which owns the database",
			},
			dbTemplateAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Computed:    true,
				Description: "The name of the template from which to create the new database",
			},
			dbEncodingAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Character set encoding to use in the new database",
			},
			dbCollationAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Collation order (LC_COLLATE) to use in the new database",
			},
			dbCTypeAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "Character classification (LC_CTYPE) to use in the new database",
			},
			dbTablespaceAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "The name of the tablespace that will be associated with the new database",
			},
			dbConnLimitAttr: {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      -1,
				Description:  "How many concurrent connections can be made to this database",
				ValidateFunc: validateConnLimit,
			},
			dbAllowConnsAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "If false then no one can connect to this database",
			},
			dbIsTemplateAttr: {
				Type:        schema.TypeBool,
				Optional:    true,
				Computed:    true,
				Description: "If true, then this database can be cloned by any user with CREATEDB privileges",
			},
		},
	}
}

func resourcePostgreSQLDatabaseCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)

	c.catalogLock.Lock()
	defer c.catalogLock.Unlock()

	conn, err := c.Connect()
	if err != nil {
		return errwrap.Wrapf("Error connecting to PostgreSQL: {{err}}", err)
	}
	defer conn.Close()

	dbName := d.Get(dbNameAttr).(string)
	b := bytes.NewBufferString("CREATE DATABASE ")
	fmt.Fprint(b, pq.QuoteIdentifier(dbName))

	//needed in order to set the owner of the db if the connection user is not a superuser
	err = grantRoleMembership(conn, d.Get(dbOwnerAttr).(string), c.username)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Error adding connection user (%q) to ROLE %q: {{err}}", c.username, d.Get(dbOwnerAttr).(string)), err)
	}
	defer func() {
		//undo the grant if the connection user is not a superuser
		err = revokeRoleMembership(conn, d.Get(dbOwnerAttr).(string), c.username)
		if err != nil {
			err = errwrap.Wrapf(fmt.Sprintf("Error removing connection user (%q) from ROLE %q: {{err}}", c.username, d.Get(dbOwnerAttr).(string)), err)
		}
	}()

	// Handle each option individually and stream results into the query
	// buffer.

	switch v, ok := d.GetOk(dbOwnerAttr); {
	case ok:
		fmt.Fprint(b, " OWNER ", pq.QuoteIdentifier(v.(string)))
	default:
		// No owner specified in the config, default to using
		// the connecting username.
		fmt.Fprint(b, " OWNER ", pq.QuoteIdentifier(c.username))
	}

	switch v, ok := d.GetOk(dbTemplateAttr); {
	case ok:
		fmt.Fprint(b, " TEMPLATE ", pq.QuoteIdentifier(v.(string)))
	case v.(string) == "", strings.ToUpper(v.(string)) != "DEFAULT":
		fmt.Fprint(b, " TEMPLATE template0")
	}

	switch v, ok := d.GetOk(dbEncodingAttr); {
	case ok:
		fmt.Fprint(b, " ENCODING ", pq.QuoteIdentifier(v.(string)))
	case v.(string) == "", strings.ToUpper(v.(string)) != "DEFAULT":
		fmt.Fprint(b, ` ENCODING "UTF8"`)
	}

	switch v, ok := d.GetOk(dbCollationAttr); {
	case ok:
		fmt.Fprint(b, " LC_COLLATE ", pq.QuoteIdentifier(v.(string)))
	case v.(string) == "", strings.ToUpper(v.(string)) != "DEFAULT":
		fmt.Fprint(b, ` LC_COLLATE "C"`)
	}

	switch v, ok := d.GetOk(dbCTypeAttr); {
	case ok:
		fmt.Fprint(b, " LC_CTYPE ", pq.QuoteIdentifier(v.(string)))
	case v.(string) == "", strings.ToUpper(v.(string)) != "DEFAULT":
		fmt.Fprint(b, ` LC_CTYPE "C"`)
	}

	if v, ok := d.GetOk(dbTablespaceAttr); ok {
		fmt.Fprint(b, " TABLESPACE ", pq.QuoteIdentifier(v.(string)))
	}

	{
		val := d.Get(dbAllowConnsAttr).(bool)
		fmt.Fprint(b, " ALLOW_CONNECTIONS ", val)
	}

	{
		val := d.Get(dbConnLimitAttr).(int)
		fmt.Fprint(b, " CONNECTION LIMIT ", val)
	}

	{
		val := d.Get(dbIsTemplateAttr).(bool)
		fmt.Fprint(b, " IS_TEMPLATE ", val)
	}

	query := b.String()
	_, err = conn.Query(query)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Error creating database %q: {{err}}", dbName), err)
	}

	d.SetId(dbName)

	// Set err outside of the return so that the deferred revoke can override err
	// if necessary.
	err = resourcePostgreSQLDatabaseReadImpl(d, meta)
	return err
}

func resourcePostgreSQLDatabaseDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	c.catalogLock.Lock()
	defer c.catalogLock.Unlock()

	conn, err := c.Connect()
	if err != nil {
		return errwrap.Wrapf("Error connecting to PostgreSQL: {{err}}", err)
	}
	defer conn.Close()

	dbName := d.Get(dbNameAttr).(string)

	if isTemplate := d.Get(dbIsTemplateAttr).(bool); isTemplate {
		// Template databases must have this attribute cleared before
		// they can be dropped.
		if err := doSetDBIsTemplate(conn, dbName, false); err != nil {
			return errwrap.Wrapf("Error updating database IS_TEMPLATE during DROP DATABASE: {{err}}", err)
		}
	}

	if err := setDBIsTemplate(conn, d); err != nil {
		return err
	}

	query := fmt.Sprintf("DROP DATABASE %s", pq.QuoteIdentifier(dbName))
	_, err = conn.Query(query)
	if err != nil {
		return errwrap.Wrapf("Error dropping database: {{err}}", err)
	}

	d.SetId("")

	return nil
}

func resourcePostgreSQLDatabaseExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	c := meta.(*Client)
	c.catalogLock.RLock()
	defer c.catalogLock.RUnlock()

	conn, err := c.Connect()
	if err != nil {
		return false, err
	}
	defer conn.Close()

	var dbName string
	err = conn.QueryRow("SELECT d.datname from pg_database d WHERE datname=$1", d.Id()).Scan(&dbName)
	switch {
	case err == sql.ErrNoRows:
		return false, nil
	case err != nil:
		return false, err
	}

	return true, nil
}

func resourcePostgreSQLDatabaseRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	c.catalogLock.RLock()
	defer c.catalogLock.RUnlock()

	return resourcePostgreSQLDatabaseReadImpl(d, meta)
}

func resourcePostgreSQLDatabaseReadImpl(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	dbId := d.Id()
	var dbName, ownerName string
	err = conn.QueryRow("SELECT d.datname, pg_catalog.pg_get_userbyid(d.datdba) from pg_database d WHERE datname=$1", dbId).Scan(&dbName, &ownerName)
	switch {
	case err == sql.ErrNoRows:
		log.Printf("[WARN] PostgreSQL database (%q) not found", dbId)
		d.SetId("")
		return nil
	case err != nil:
		return errwrap.Wrapf("Error reading database: {{err}}", err)
	}

	var dbEncoding, dbCollation, dbCType, dbTablespaceName string
	var dbConnLimit int
	var dbAllowConns, dbIsTemplate bool
	err = conn.QueryRow(`SELECT pg_catalog.pg_encoding_to_char(d.encoding), d.datcollate, d.datctype, ts.spcname, d.datconnlimit, d.datallowconn, d.datistemplate FROM pg_catalog.pg_database AS d, pg_catalog.pg_tablespace AS ts WHERE d.datname = $1 AND d.dattablespace = ts.oid`, dbId).
		Scan(
			&dbEncoding, &dbCollation, &dbCType, &dbTablespaceName,
			&dbConnLimit, &dbAllowConns, &dbIsTemplate,
		)
	switch {
	case err == sql.ErrNoRows:
		log.Printf("[WARN] PostgreSQL database (%q) not found", dbId)
		d.SetId("")
		return nil
	case err != nil:
		return errwrap.Wrapf("Error reading database: {{err}}", err)
	default:
		d.Set(dbNameAttr, dbName)
		d.Set(dbOwnerAttr, ownerName)
		d.Set(dbEncodingAttr, dbEncoding)
		d.Set(dbCollationAttr, dbCollation)
		d.Set(dbCTypeAttr, dbCType)
		d.Set(dbTablespaceAttr, dbTablespaceName)
		d.Set(dbConnLimitAttr, dbConnLimit)
		d.Set(dbAllowConnsAttr, dbAllowConns)
		d.Set(dbIsTemplateAttr, dbIsTemplate)
		dbTemplate := d.Get(dbTemplateAttr).(string)
		if dbTemplate == "" {
			dbTemplate = "template0"
		}
		d.Set(dbTemplateAttr, dbTemplate)
		return nil
	}
}

func resourcePostgreSQLDatabaseUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	c.catalogLock.Lock()
	defer c.catalogLock.Unlock()

	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := setDBName(conn, d); err != nil {
		return err
	}

	if err := setDBOwner(c, conn, d); err != nil {
		return err
	}

	if err := setDBTablespace(conn, d); err != nil {
		return err
	}

	if err := setDBConnLimit(conn, d); err != nil {
		return err
	}

	if err := setDBAllowConns(conn, d); err != nil {
		return err
	}

	if err := setDBIsTemplate(conn, d); err != nil {
		return err
	}

	// Empty values: ALTER DATABASE name RESET configuration_parameter;

	return resourcePostgreSQLDatabaseReadImpl(d, meta)
}

func setDBName(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(dbNameAttr) {
		return nil
	}

	oraw, nraw := d.GetChange(dbNameAttr)
	o := oraw.(string)
	n := nraw.(string)
	if n == "" {
		return errors.New("Error setting database name to an empty string")
	}

	query := fmt.Sprintf("ALTER DATABASE %s RENAME TO %s", pq.QuoteIdentifier(o), pq.QuoteIdentifier(n))
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating database name: {{err}}", err)
	}
	d.SetId(n)

	return nil
}

func setDBOwner(c *Client, conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(dbOwnerAttr) {
		return nil
	}

	owner := d.Get(dbOwnerAttr).(string)
	if owner == "" {
		return nil
	}

	//needed in order to set the owner of the db if the connection user is not a superuser
	err := grantRoleMembership(conn, d.Get(dbOwnerAttr).(string), c.username)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Error adding connection user (%q) to ROLE %q: {{err}}", c.username, d.Get(dbOwnerAttr).(string)), err)
	}
	defer func() {
		// undo the grant if the connection user is not a superuser
		err = revokeRoleMembership(conn, d.Get(dbOwnerAttr).(string), c.username)
		if err != nil {
			err = errwrap.Wrapf(fmt.Sprintf("Error removing connection user (%q) from ROLE %q: {{err}}", c.username, d.Get(dbOwnerAttr).(string)), err)
		}
	}()

	dbName := d.Get(dbNameAttr).(string)
	query := fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", pq.QuoteIdentifier(dbName), pq.QuoteIdentifier(owner))
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating database OWNER: {{err}}", err)
	}

	return err
}

func setDBTablespace(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(dbTablespaceAttr) {
		return nil
	}

	tbspName := d.Get(dbTablespaceAttr).(string)
	dbName := d.Get(dbNameAttr).(string)
	var query string
	if tbspName == "" || strings.ToUpper(tbspName) == "DEFAULT" {
		query = fmt.Sprintf("ALTER DATABASE %s RESET TABLESPACE", pq.QuoteIdentifier(dbName))
	} else {
		query = fmt.Sprintf("ALTER DATABASE %s SET TABLESPACE %s", pq.QuoteIdentifier(dbName), pq.QuoteIdentifier(tbspName))
	}

	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating database TABLESPACE: {{err}}", err)
	}

	return nil
}

func setDBConnLimit(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(dbConnLimitAttr) {
		return nil
	}

	connLimit := d.Get(dbConnLimitAttr).(int)
	dbName := d.Get(dbNameAttr).(string)
	query := fmt.Sprintf("ALTER DATABASE %s CONNECTION LIMIT = %d", pq.QuoteIdentifier(dbName), connLimit)
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating database CONNECTION LIMIT: {{err}}", err)
	}

	return nil
}

func setDBAllowConns(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(dbAllowConnsAttr) {
		return nil
	}

	allowConns := d.Get(dbAllowConnsAttr).(bool)
	dbName := d.Get(dbNameAttr).(string)
	query := fmt.Sprintf("ALTER DATABASE %s ALLOW_CONNECTIONS %t", pq.QuoteIdentifier(dbName), allowConns)
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating database ALLOW_CONNECTIONS: {{err}}", err)
	}

	return nil
}

func setDBIsTemplate(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(dbIsTemplateAttr) {
		return nil
	}

	if err := doSetDBIsTemplate(conn, d.Get(dbNameAttr).(string), d.Get(dbIsTemplateAttr).(bool)); err != nil {
		return errwrap.Wrapf("Error updating database IS_TEMPLATE: {{err}}", err)
	}

	return nil
}

func doSetDBIsTemplate(conn *sql.DB, dbName string, isTemplate bool) error {
	query := fmt.Sprintf("ALTER DATABASE %s IS_TEMPLATE %t", pq.QuoteIdentifier(dbName), isTemplate)
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating database IS_TEMPLATE: {{err}}", err)
	}

	return nil
}

func grantRoleMembership(conn *sql.DB, dbOwner string, connUsername string) error {
	if dbOwner != "" && dbOwner != connUsername {
		query := fmt.Sprintf("GRANT %s TO %s", pq.QuoteIdentifier(dbOwner), pq.QuoteIdentifier(connUsername))
		_, err := conn.Query(query)
		if err != nil {
			// is already member or role
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				return nil
			}
			return errwrap.Wrapf("Error granting membership: {{err}}", err)
		}
	}
	return nil
}

func revokeRoleMembership(conn *sql.DB, dbOwner string, connUsername string) error {
	if dbOwner != "" && dbOwner != connUsername {
		query := fmt.Sprintf("REVOKE %s FROM %s", pq.QuoteIdentifier(dbOwner), pq.QuoteIdentifier(connUsername))
		_, err := conn.Query(query)
		if err != nil {
			return errwrap.Wrapf("Error revoking membership: {{err}}", err)
		}
	}
	return nil
}
