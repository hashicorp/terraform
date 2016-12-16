package influxdb

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/influxdata/influxdb/client"
)

func resourceRetentionPolicy() *schema.Resource {
	return &schema.Resource{
		Create: createRetentionPolicy,
		Read:   readRetentionPolicy,
		Update: updateRetentionPolicy,
		Delete: deleteRetentionPolicy,

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
			"duration": &schema.Schema{
				Type:     schema.TypeString,
				Required: false,
				Optional: true,
				Default:  "INF",
			},
			"is_default": &schema.Schema{
				Type:     schema.TypeBool,
				Required: false,
				Optional: true,
				Default:  false,
			},
			"replication": &schema.Schema{
				Type:     schema.TypeInt,
				Required: false,
				Optional: true,
				Default:  1,
			},
			"shard_duration": &schema.Schema{
				Type:     schema.TypeString,
				Required: false,
				Optional: true,
				Default:  "",
			},
		},
	}
}

func createRetentionPolicy(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)

	name := d.Get("name").(string)
	database := d.Get("database").(string)
	duration := fmt.Sprintf("DURATION %s", d.Get("duration").(string))
	replication := fmt.Sprintf("REPLICATION %d", d.Get("replication").(int))

	shardDuration := ""
	if len(d.Get("shard_duration").(string)) > 0 {
		shardDuration = fmt.Sprintf("SHARD DURATION %s", d.Get("shard_duration").(string))
	}

	isDefault := ""
	if d.Get("is_default").(bool) {
		isDefault = "DEFAULT"
	}

	queryStr := fmt.Sprintf("CREATE RETENTION POLICY %s ON %s %s %s %s %s", quoteIdentifier(name), quoteIdentifier(database), duration, replication, shardDuration, isDefault)
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

	d.Set("database", database)
	d.Set("duration", d.Get("duration").(string))
	d.Set("is_default", d.Get("is_default").(bool))
	d.Set("name", name)
	d.Set("replication", d.Get("replication").(int))
	d.Set("shard_duration", d.Get("shard_duration").(string))
	d.SetId(fmt.Sprintf("influxdb-rp:%s", name))

	return nil
}

func deleteRetentionPolicy(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)
	name := d.Get("name").(string)
	database := d.Get("database").(string)

	queryStr := fmt.Sprintf("DROP RETENTION POLICY %s ON %s", quoteIdentifier(name), quoteIdentifier(database))
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

func readRetentionPolicy(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)
	name := d.Get("name").(string)
	database := d.Get("database").(string)

	queryStr := fmt.Sprintf("SHOW RETENTION POLICIES ON %s", quoteIdentifier(database))
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

	for _, series := range resp.Results[0].Series {
		for _, result := range series.Values {
			if result[0].(string) == name {
				return nil
			}
		}
	}

	d.SetId("")

	return nil
}

func updateRetentionPolicy(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*client.Client)
	name := d.Get("name").(string)
	database := d.Get("database").(string)
	duration := fmt.Sprintf("DURATION %s", d.Get("duration").(string))
	replication := fmt.Sprintf("REPLICATION %d", d.Get("replication").(int))

	shardDuration := ""
	if len(d.Get("shard_duration").(string)) > 0 {
		shardDuration = fmt.Sprintf("SHARD DURATION %s", d.Get("shard_duration").(string))
	}

	isDefault := ""
	if d.Get("is_default").(bool) {
		isDefault = "DEFAULT"
	}

	queryStr := fmt.Sprintf("ALTER RETENTION POLICY %s ON %s %s %s %s %s", quoteIdentifier(name), quoteIdentifier(database), duration, replication, shardDuration, isDefault)
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

	d.Set("database", database)
	d.Set("duration", d.Get("duration").(string))
	d.Set("is_default", d.Get("is_default").(bool))
	d.Set("name", name)
	d.Set("replication", d.Get("replication").(int))
	d.Set("shard_duration", d.Get("shard_duration").(string))
	d.SetId(fmt.Sprintf("influxdb-rp:%s", name))

	return nil
}
