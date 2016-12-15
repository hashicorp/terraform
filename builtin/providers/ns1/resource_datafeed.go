package ns1

import (
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	ns1 "gopkg.in/ns1/ns1-go.v2/rest"
	"gopkg.in/ns1/ns1-go.v2/rest/model/data"
)

func dataFeedResource() *schema.Resource {
	return &schema.Resource{
		Create: resourceNS1DataFeedCreate,
		Read:   resourceNS1DataFeedRead,
		Update: resourceNS1DataFeedUpdate,
		Delete: resourceNS1DataFeedDelete,
		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"config": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
			"source_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceNS1DataFeedCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	f := buildNS1DataFeed(d)

	log.Printf("[INFO] Creating NS1 data feed: %s \n", f.Name)

	if _, err := client.DataFeeds.Create(d.Get("source_id").(string), f); err != nil {
		return err
	}

	d.SetId(f.ID)

	return resourceNS1DataFeedRead(d, meta)
}

func resourceNS1DataFeedRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Reading NS1 data feed: %s \n", d.Id())

	f, _, err := client.DataFeeds.Get(d.Get("source_id").(string), d.Id())
	if err != nil {
		return err
	}

	d.Set("name", f.Name)
	d.Set("config", f.Config)

	return nil
}

func resourceNS1DataFeedUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	f := buildNS1DataFeed(d)
	f.ID = d.Id()

	log.Printf("[INFO] Updating NS1 data feed: %s \n", f.ID)

	if _, err := client.DataFeeds.Update(d.Get("source_id").(string), f); err != nil {
		return err
	}

	return nil
}

func resourceNS1DataFeedDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ns1.Client)

	log.Printf("[INFO] Deleting NS1 data feed: %s \n", d.Id())

	if _, err := client.DataFeeds.Delete(d.Get("source_id").(string), d.Id()); err != nil {
		return err
	}

	d.SetId("")

	return nil
}

func buildNS1DataFeed(d *schema.ResourceData) *data.Feed {
	return &data.Feed{
		Name:     d.Get("name").(string),
		Config:   d.Get("config").(map[string]interface{}),
		SourceID: d.Get("source_id").(string),
	}
}
