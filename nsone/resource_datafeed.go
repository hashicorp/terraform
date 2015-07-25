package nsone

import (
	"github.com/bobtfish/go-nsone-api"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataFeedResource() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"source_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
		Create: DataFeedCreate,
		Read:   DataFeedRead,
		Update: DataFeedUpdate,
		Delete: DataFeedDelete,
	}
}

func dataFeedToResourceData(d *schema.ResourceData, df *nsone.DataFeed) {
	d.SetId(df.Id)
	d.Set("name", df.Name)
}

func DataFeedCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	df := nsone.NewDataFeed(d.Get("source_id").(string))
	df.Name = d.Get("name").(string)
	err := client.CreateDataFeed(df)
	if err != nil {
		return err
	}
	dataFeedToResourceData(d, df)
	return nil
}

func DataFeedRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	df, err := client.GetDataFeed(d.Get("source_id").(string), d.Id())
	if err != nil {
		return err
	}
	dataFeedToResourceData(d, df)
	return nil
}

func DataFeedDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*nsone.APIClient)
	err := client.DeleteDataFeed(d.Get("source_id").(string), d.Id())
	d.SetId("")
	return err
}

func DataFeedUpdate(d *schema.ResourceData, meta interface{}) error {
	panic("Update not implemented")
	return nil
}
