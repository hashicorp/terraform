package postgresql

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lib/pq"
)

func resourcePostgreSQLDatabase() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgreSQLDatabaseCreate,
		Read:   resourcePostgreSQLDatabaseRead,
		Update: resourcePostgreSQLDatabaseUpdate,
		Delete: resourcePostgreSQLDatabaseDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The PostgreSQL database name to connect to",
			},
			"owner": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The role name of the user who will own the new database",
			},
			"template": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the template from which to create the new database.",
			},
			"encoding": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Character set encoding to use in the new database.",
			},
			"lc_collate": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Collation order (LC_COLLATE) to use in the new database.",
			},
			"lc_ctype": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Character classification (LC_CTYPE) to use in the new database.",
			},
			"tablespace_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The name of the tablespace that will be associated with the new database.",
			},
			"connection_limit": {
				Type:         schema.TypeInt,
				Optional:     true,
				Computed:     true,
				Description:  "How many concurrent connections can be made to this database",
				ValidateFunc: validateConnLimit,
			},
			"allow_connections": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "If false then no one can connect to this database.",
			},
			"is_template": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "If true, then this database can be cloned by any user with CREATEDB privileges.",
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
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return errwrap.Wrapf("Error connecting to PostgreSQL: {{err}}", err)
	}
	defer conn.Close()

	connUsername := client.username

	const numOptions = 9
	createOpts := make([]string, 0, numOptions)

	stringOpts := []struct {
		hclKey string
		sqlKey string
	}{
		{"owner", "OWNER"},
		{"template", "TEMPLATE"},
		{"encoding", "ENCODING"},
		{"lc_collate", "LC_COLLATE"},
		{"lc_ctype", "LC_CTYPE"},
		{"tablespace_name", "TABLESPACE"},
	}
	for _, opt := range stringOpts {
		v, ok := d.GetOk(opt.hclKey)
		var val string
		if !ok {
			// Set the owner to the connection username
			if opt.hclKey == "owner" && v.(string) == "" {
				val = connUsername
			} else {
				continue
			}
		}

		val = v.(string)

		// Set the owner to the connection username
		if opt.hclKey == "owner" && val == "" {
			val = connUsername
		}

		if val != "" {
			createOpts = append(createOpts, fmt.Sprintf("%s=%s", opt.sqlKey, pq.QuoteIdentifier(val)))
		}
	}

	intOpts := []struct {
		hclKey string
		sqlKey string
	}{
		{"connection_limit", "CONNECTION LIMIT"},
	}
	for _, opt := range intOpts {
		v, ok := d.GetOk(opt.hclKey)
		if !ok {
			continue
		}

		val := v.(int)
		createOpts = append(createOpts, fmt.Sprintf("%s=%d", opt.sqlKey, val))
	}

	boolOpts := []struct {
		hclKey string
		sqlKey string
	}{
		{"allow_connections", "ALLOW_CONNECTIONS"},
		{"is_template", "IS_TEMPLATE"},
	}
	for _, opt := range boolOpts {
		v, ok := d.GetOk(opt.hclKey)
		if !ok {
			continue
		}

		val := v.(bool)
		createOpts = append(createOpts, fmt.Sprintf("%s=%t", opt.sqlKey, val))
	}

	dbName := d.Get("name").(string)
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
	client := meta.(*Client)
	conn, err := client.Connect()
	if err != nil {
		return errwrap.Wrapf("Error connecting to PostgreSQL: {{err}}", err)
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
		return errwrap.Wrapf("Error dropping database: {{err}}", err)
	}

	d.SetId("")

	return nil
}

func resourcePostgreSQLDatabaseRead(d *schema.ResourceData, meta interface{}) error {
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
		return errwrap.Wrapf("Error reading database: {{err}}", err)
	default:
		d.Set("owner", owner)
		return nil
	}
}

func resourcePostgreSQLDatabaseUpdate(d *schema.ResourceData, meta interface{}) error {
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
				return errwrap.Wrapf("Error updating owner: {{err}}", err)
			}
		}
	}

	return resourcePostgreSQLDatabaseRead(d, meta)
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
			return errwrap.Wrapf("Error granting membership: {{err}}", err)
		}
	}
	return nil
}
