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
				Description: "The role name of the user who will own the new database",
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
				Computed:     true,
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

func validateConnLimit(v interface{}, key string) (warnings []string, errors []error) {
	value := v.(int)
	if value < -1 {
		errors = append(errors, fmt.Errorf("%d can not be less than -1", key))
	}
	return
}

func resourcePostgreSQLDatabaseCreate(d *schema.ResourceData, meta interface{}) error {
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
		{dbOwnerAttr, "OWNER"},
		{dbTemplateAttr, "TEMPLATE"},
		{dbEncodingAttr, "ENCODING"},
		{dbCollationAttr, "LC_COLLATE"},
		{dbCTypeAttr, "LC_CTYPE"},
		{dbTablespaceAttr, "TABLESPACE"},
	}
	intOpts := []struct {
		hclKey string
		sqlKey string
	}{
		{dbConnLimitAttr, "CONNECTION LIMIT"},
	}
	boolOpts := []struct {
		hclKey string
		sqlKey string
	}{
		{dbAllowConnsAttr, "ALLOW_CONNECTIONS"},
		{dbIsTemplateAttr, "IS_TEMPLATE"},
	}

	createOpts := make([]string, 0, len(stringOpts)+len(intOpts)+len(boolOpts))

	for _, opt := range stringOpts {
		v, ok := d.GetOk(opt.hclKey)
		var val string
		if !ok {
			switch {
			case opt.hclKey == dbOwnerAttr && v.(string) == "":
				// No owner specified in the config, default to using
				// the connecting username.
				val = c.username
			case strings.ToUpper(v.(string)) == "DEFAULT" &&
				(opt.hclKey == dbTemplateAttr ||
					opt.hclKey == dbEncodingAttr ||
					opt.hclKey == dbCollationAttr ||
					opt.hclKey == dbCTypeAttr):

				// Use the defaults from the template database
				// as opposed to best practices.
				fallthrough
			default:
				continue
			}
		}

		val = v.(string)

		switch {
		case opt.hclKey == dbOwnerAttr && (val == "" || strings.ToUpper(val) == "DEFAULT"):
			// Owner was blank/DEFAULT, default to using the connecting username.
			val = c.username
			d.Set(dbOwnerAttr, val)
		case opt.hclKey == dbTablespaceAttr && (val == "" || strings.ToUpper(val) == "DEFAULT"):
			val = "pg_default"
			d.Set(dbTablespaceAttr, val)
		case opt.hclKey == dbTemplateAttr:
			if val == "" {
				val = "template0"
				d.Set(dbTemplateAttr, val)
			} else if strings.ToUpper(val) == "DEFAULT" {
				val = ""
			}
		case opt.hclKey == dbEncodingAttr:
			if val == "" {
				val = "UTF8"
				d.Set(dbEncodingAttr, val)
			} else if strings.ToUpper(val) == "DEFAULT" {
				val = ""
			}
		case opt.hclKey == dbCollationAttr:
			if val == "" {
				val = "C"
				d.Set(dbCollationAttr, val)
			} else if strings.ToUpper(val) == "DEFAULT" {
				val = ""
			}
		case opt.hclKey == dbCTypeAttr:
			if val == "" {
				val = "C"
				d.Set(dbCTypeAttr, val)
			} else if strings.ToUpper(val) == "DEFAULT" {
				val = ""
			}
		}

		if val != "" {
			createOpts = append(createOpts, fmt.Sprintf("%s=%s", opt.sqlKey, pq.QuoteIdentifier(val)))
		}
	}

	for _, opt := range intOpts {
		v, ok := d.GetOk(opt.hclKey)
		if !ok {
			continue
		}

		val := v.(int)
		createOpts = append(createOpts, fmt.Sprintf("%s=%d", opt.sqlKey, val))
	}

	for _, opt := range boolOpts {
		v := d.Get(opt.hclKey)

		valStr := "FALSE"
		if val := v.(bool); val {
			valStr = "TRUE"
		}
		createOpts = append(createOpts, fmt.Sprintf("%s=%s", opt.sqlKey, valStr))
	}

	dbName := d.Get(dbNameAttr).(string)
	createStr := strings.Join(createOpts, " ")
	if len(createOpts) > 0 {
		createStr = " WITH " + createStr
	}
	query := fmt.Sprintf("CREATE DATABASE %s%s", pq.QuoteIdentifier(dbName), createStr)
	_, err = conn.Query(query)
	if err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Error creating database %s: {{err}}", dbName), err)
	}

	d.SetId(dbName)

	return resourcePostgreSQLDatabaseRead(d, meta)
}

func resourcePostgreSQLDatabaseDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
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

func resourcePostgreSQLDatabaseRead(d *schema.ResourceData, meta interface{}) error {
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
		log.Printf("[WARN] PostgreSQL database (%s) not found", d.Id())
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
		log.Printf("[WARN] PostgreSQL database (%s) not found", d.Id())
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
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := setDBName(conn, d); err != nil {
		return err
	}

	if err := setDBOwner(conn, d); err != nil {
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

	return resourcePostgreSQLDatabaseRead(d, meta)
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

func setDBOwner(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(dbOwnerAttr) {
		return nil
	}

	owner := d.Get(dbOwnerAttr).(string)
	if owner == "" {
		return nil
	}

	dbName := d.Get(dbNameAttr).(string)
	query := fmt.Sprintf("ALTER DATABASE %s OWNER TO %s", pq.QuoteIdentifier(dbName), pq.QuoteIdentifier(owner))
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating database OWNER: {{err}}", err)
	}

	return nil
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
