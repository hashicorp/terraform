package ns1

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/data"
)

func dataSourceResource() *schema.Resource {
	return &schema.Resource{
		Create: resourceNS1DataSourceCreate,
		Read:   resourceNS1DataSourceRead,
		Update: resourceNS1DataSourceUpdate,
		Delete: resourceNS1DataSourceDelete,
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"config": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

func resourceNS1DataSourceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	s := buildNS1DataSource(d)

	log.Printf("[INFO] Creating NS1 data source: %s \n", s.Name)

	if _, err := client.DataSources.Create(s); err != nil {
		return err
	}

	d.SetId(s.ID)

	return resourceNS1DataSourceRead(d, meta)
}

func resourceNS1DataSourceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Reading NS1 data source: %s \n", d.Id())

	s, _, err := client.DataSources.Get(d.Id())
	if err != nil {
		return err
	}

	d.Set("name", s.Name)
	d.Set("type", s.Type)
	d.Set("config", s.Config)

	return nil
}

func resourceNS1DataSourceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	s := buildNS1DataSource(d)
	s.ID = d.Id()

	log.Printf("[INFO] Updating NS1 data source: %s \n", s.ID)

	if _, err := client.DataSources.Update(s); err != nil {
		return err
	}

	return nil
}

func resourceNS1DataSourceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Deleting NS1 data source: %s \n", d.Id())

	if _, err := client.DataSources.Delete(d.Id()); err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func buildNS1DataSource(d *schema.ResourceData) *data.Source {
	s := data.NewSource(d.Get("name").(string), d.Get("type").(string))
	s.Config = d.Get("config").(map[string]interface{})
	return s
}
