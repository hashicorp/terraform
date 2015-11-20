package google

import (
	"google.golang.org/api/bigquery/v2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceBigQueryTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceBigQueryTableCreate,
		Read:   resourceBigQueryTableRead,
		Delete: resourceBigQueryTableDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"datasetId": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"can_delete": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default: false,
			},
		},
	}
}

func resourceBigQueryTableCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	datasetId := d.Get("datasetId").(string)
	tableName := d.Get("name").(string)
	tRef := &bigquery.TableReference{DatasetId: datasetId, ProjectId: config.Project, TableId: tableName}
	table := &bigquery.Table{TableReference: tRef}

	call := config.clientBigQuery.Tables.Insert(config.Project, datasetId, table)
	_, err := call.Do()
	if err != nil {
		return err
	}
	
	err = resourceBigQueryTableRead(d, meta)
	if err != nil {
		return err
	}

	return nil
}

func resourceBigQueryTableRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	
	call := config.clientBigQuery.Tables.Get(config.Project, d.Get("datasetId").(string), d.Get("name").(string))
	res, err := call.Do()
	if err != nil {
		return err
	}

	d.SetId(res.Id)
	return nil
}


func resourceBigQueryTableDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	if d.Get("can_delete").(bool) == true {
		call := config.clientBigQuery.Tables.Delete(config.Project, d.Get("datasetId").(string), d.Get("name").(string))
		err := call.Do()
		if err != nil {
			return err
		}
	}

	d.SetId("")	
	return nil
}
