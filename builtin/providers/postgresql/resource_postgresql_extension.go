package postgresql

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lib/pq"
)

func resourcePostgreSQLExtension() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgreSQLExtensionCreate,
		Read:   resourcePostgreSQLExtensionRead,
		Delete: resourcePostgreSQLExtensionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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

	extensionName := d.Get("name").(string)

	query := fmt.Sprintf("CREATE EXTENSION %s", pq.QuoteIdentifier(extensionName))
	_, err = conn.Query(query)
	if err != nil {
		return errwrap.Wrapf("Error creating extension: {{err}}", err)
	}

	d.SetId(extensionName)

	return resourcePostgreSQLExtensionRead(d, meta)
}

func resourcePostgreSQLExtensionRead(d *schema.ResourceData, meta interface{}) error {
	c := meta.(*Client)
	conn, err := c.Connect()
	if err != nil {
		return err
	}
	defer conn.Close()

	dbId := d.Id()
	extensionName := d.Get("name").(string)

	var hasExtension bool
	err = conn.QueryRow("SELECT TRUE from pg_catalog.pg_extension d WHERE extname=$1", dbId).Scan(&hasExtension)
	switch {
	case err == sql.ErrNoRows:
		log.Printf("[WARN] PostgreSQL extension (%s) not found", d.Id())
		d.SetId("")
		return nil
	case err != nil:
		return errwrap.Wrapf("Error reading extension: {{err}}", err)
	default:
		d.Set("extension", hasExtension)
		d.SetId(extensionName)
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

	extensionName := d.Get("name").(string)

	query := fmt.Sprintf("DROP EXTENSION %s", pq.QuoteIdentifier(extensionName))
	_, err = conn.Query(query)
	if err != nil {
		return errwrap.Wrapf("Error deleting extension: {{err}}", err)
	}

	d.SetId("")

	return nil
}
