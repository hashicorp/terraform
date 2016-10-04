package postgresql

import (
	"database/sql"
	"fmt"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/lib/pq"
)

func resourcePostgreSQLExtension() *schema.Resource {
	return &schema.Resource{
		Create: resourcePostgreSQLExtensionCreate,
		Delete: resourcePostgreSQLExtensionDelete,

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
	client := meta.(*Client)
	conn, err := client.Connect()
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

	return resourcePostgreSQLRoleRead(d, meta)
}

func resourcePostgreSQLExtensionDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*Client)
	conn, err := client.Connect()
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
