package influxdb

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/influxdata/influxdb/client"
)

func resourceDatabase() *schema.Resource {
	return &schema.Resource{
		Create: createDatabase,
		Read:   readDatabase,
		Delete: deleteDatabase,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func createDatabase(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)

	name := d.Get("name").(string)
	queryStr := fmt.Sprintf("CREATE DATABASE %s", quoteIdentifier(name))
	query := client.Query{
		Command: queryStr,
	}

	resp, err := conn.Query(query)
	if err != nil {
		return err
	}
	if resp.Err != nil {
		return resp.Err
	}

	d.SetId(name)

	return nil
}

func readDatabase(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)
	name := d.Id()

	// InfluxDB doesn't have a command to check the existence of a single
	// database, so we instead must read the list of all databases and see
	// if ours is present in it.
	query := client.Query{
		Command: "SHOW DATABASES",
	}

	resp, err := conn.Query(query)
	if err != nil {
		return err
	}
	if resp.Err != nil {
		return resp.Err
	}

	for _, result := range resp.Results[0].Series[0].Values {
		if result[0] == name {
			return nil
		}
	}

	// If we fell out here then we didn't find our database in the list.
	d.SetId("")

	return nil
}

func deleteDatabase(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)
	name := d.Id()

	queryStr := fmt.Sprintf("DROP DATABASE %s", quoteIdentifier(name))
	query := client.Query{
		Command: queryStr,
	}

	resp, err := conn.Query(query)
	if err != nil {
		return err
	}
	if resp.Err != nil {
		return resp.Err
	}

	d.SetId("")

	return nil
}
