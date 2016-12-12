package postgresql

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lib/pq"
)

const (
	extNameAttr    = "name"
	extSchemaAttr  = "schema"
	extVersionAttr = "version"
)

func resourcePostgreSQLExtension() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgreSQLExtensionCreate,
		Read:   resourcePostgreSQLExtensionRead,
		Update: resourcePostgreSQLExtensionUpdate,
		Delete: resourcePostgreSQLExtensionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			extNameAttr: {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			extSchemaAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Sets the schema of an extension",
			},
			extVersionAttr: {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				Description: "Sets the version number of the extension",
			},
		},
	}
}

func resourcePostgreSQLExtensionCreate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	extName := d.Get(extNameAttr).(string)

	b := bytes.NewBufferString("CREATE EXTENSION ")
	fmt.Fprintf(b, pq.QuoteIdentifier(extName))

	if v, ok := d.GetOk(extSchemaAttr); ok {
		fmt.Fprint(b, " SCHEMA ", pq.QuoteIdentifier(v.(string)))
	}

	if v, ok := d.GetOk(extVersionAttr); ok {
		fmt.Fprint(b, " VERSION ", pq.QuoteIdentifier(v.(string)))
	}

	query := b.String()
	_, err = conn.Query(query)
	if err != nil {
		return errwrap.Wrapf("Error creating extension: {{err}}", err)
	}

	d.SetId(extName)

	return resourcePostgreSQLExtensionRead(d, meta)
}

func resourcePostgreSQLExtensionRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	extID := d.Id()
	var extName, extSchema, extVersion string
	err = conn.QueryRow("SELECT e.extname, n.nspname, e.extversion FROM pg_catalog.pg_extension e, pg_catalog.pg_namespace n WHERE n.oid = e.extnamespace AND e.extname = $1", extID).Scan(&extName, &extSchema, &extVersion)
	switch {
	case err == sql.ErrNoRows:
		log.Printf("[WARN] PostgreSQL extension (%s) not found", d.Id())
		d.SetId("")
		return nil
	case err != nil:
		return errwrap.Wrapf("Error reading extension: {{err}}", err)
	default:
		d.Set(extNameAttr, extName)
		d.Set(extSchemaAttr, extSchema)
		d.Set(extVersionAttr, extVersion)
		d.SetId(extName)
		return nil
	}
}

func resourcePostgreSQLExtensionDelete(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	extID := d.Id()

	query := fmt.Sprintf("DROP EXTENSION %s", pq.QuoteIdentifier(extID))
	_, err = conn.Query(query)
	if err != nil {
		return errwrap.Wrapf("Error deleting extension: {{err}}", err)
	}

	d.SetId("")

	return nil
}

func resourcePostgreSQLExtensionUpdate(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	// Can't rename a schema

	if err := setExtSchema(conn, d); err != nil {
		return err
	}

	if err := setExtVersion(conn, d); err != nil {
		return err
	}

	return resourcePostgreSQLExtensionRead(d, meta)
}

func setExtSchema(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(extSchemaAttr) {
		return nil
	}

	extID := d.Id()
	_, nraw := d.GetChange(extSchemaAttr)
	n := nraw.(string)
	if n == "" {
		return errors.New("Error setting extension name to an empty string")
	}

	query := fmt.Sprintf("ALTER EXTENSION %s SET SCHEMA %s", pq.QuoteIdentifier(extID), pq.QuoteIdentifier(n))
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating extension SCHEMA: {{err}}", err)
	}

	return nil
}

func setExtVersion(conn *sql.DB, d *schema.ResourceData) error {
	if !d.HasChange(extVersionAttr) {
		return nil
	}

	extID := d.Id()

	b := bytes.NewBufferString("ALTER EXTENSION ")
	fmt.Fprintf(b, "%s UPDATE", pq.QuoteIdentifier(extID))

	_, nraw := d.GetChange(extVersionAttr)
	n := nraw.(string)
	if n != "" {
		fmt.Fprintf(b, " TO %s", pq.QuoteIdentifier(n))
	}

	query := b.String()
	if _, err := conn.Query(query); err != nil {
		return errwrap.Wrapf("Error updating extension version: {{err}}", err)
	}

	return nil
}
