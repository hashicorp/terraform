package google

import (
	"google.golang.org/api/bigquery/v2"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceBigQueryDataset() *schema.Resource {
	return &schema.Resource{
		Create: resourceBigQueryDatasetCreate,
		Read:   resourceBigQueryDatasetRead,
		Update: resourceBigQueryDatasetUpdate,
		Delete: resourceBigQueryDatasetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
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

func resourceBigQueryDatasetCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	datasetName := d.Get("name").(string)
	dRef := &bigquery.DatasetReference{DatasetId: datasetName, ProjectId: config.Project}
	dataset := &bigquery.Dataset{DatasetReference: dRef}
	if d.Get("FriendlyName") != nil {
		dataset.FriendlyName = d.Get("FriendlyName").(string)
	}

	call := config.clientBigQuery.Datasets.Insert(config.Project, dataset)
	_, err := call.Do()
	if err != nil {
		return err
	}
	
	err = resourceBigQueryDatasetRead(d, meta)
	if err != nil {
		return err
	}

	return nil
}

func resourceBigQueryDatasetRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	
	call := config.clientBigQuery.Datasets.Get(config.Project, d.Get("name").(string))
	res, err := call.Do()
	if err != nil {
		return err
	}

	d.SetId(res.Id)
	return nil
}

//  basically a no-op from terraform's point of view.  but it allows can_delete
//  setting to be altered
func resourceBigQueryDatasetUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceBigQueryDatasetDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	if d.Get("can_delete").(bool) == true {
		call := config.clientBigQuery.Datasets.Delete(config.Project, d.Get("name").(string))
		err := call.Do()
		if err != nil {
			return err
		}
	}

	d.SetId("")	
	return nil
}
