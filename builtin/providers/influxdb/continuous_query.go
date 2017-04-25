package influxdb

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/influxdata/influxdb/client"
)

func resourceContinuousQuery() *schema.Resource {
	return &schema.Resource{
		Create: createContinuousQuery,
		Read:   readContinuousQuery,
		Delete: deleteContinuousQuery,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"database": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"query": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func createContinuousQuery(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)

	name := d.Get("name").(string)
	database := d.Get("database").(string)

	queryStr := fmt.Sprintf("CREATE CONTINUOUS QUERY %s ON %s BEGIN %s END", name, quoteIdentifier(database), d.Get("query").(string))
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

	d.Set("name", name)
	d.Set("database", database)
	d.Set("query", d.Get("query").(string))
	d.SetId(fmt.Sprintf("influxdb-cq:%s", name))

	return readContinuousQuery(d, meta)
}

func readContinuousQuery(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)
	name := d.Get("name").(string)
	database := d.Get("database").(string)

	// InfluxDB doesn't have a command to check the existence of a single
	// ContinuousQuery, so we instead must read the list of all ContinuousQuerys and see
	// if ours is present in it.
	query := client.Query{
		Command: "SHOW CONTINUOUS QUERIES",
	}

	resp, err := conn.Query(query)
	if err != nil {
		return err
	}
	if resp.Err != nil {
		return resp.Err
	}

	for _, series := range resp.Results[0].Series {
		if series.Name == database {
			for _, result := range series.Values {
				if result[0].(string) == name {
					return nil
				}
			}
		}
	}

	// If we fell out here then we didn't find our ContinuousQuery in the list.
	d.SetId("")

	return nil
}

func deleteContinuousQuery(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)
	name := d.Get("name").(string)
	database := d.Get("database").(string)

	queryStr := fmt.Sprintf("DROP CONTINUOUS QUERY %s ON %s", name, quoteIdentifier(database))
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
