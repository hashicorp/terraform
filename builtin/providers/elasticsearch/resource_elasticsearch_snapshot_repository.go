package elasticsearch

import (
	"context"

	"github.com/hashicorp/terraform/helper/schema"
	elastic "gopkg.in/olivere/elastic.v5"
)

func resourceElasticsearchSnapshotRepository() *schema.Resource {
	return &schema.Resource{
		Create: resourceElasticsearchSnapshotRepositoryCreate,
		Read:   resourceElasticsearchSnapshotRepositoryRead,
		Update: resourceElasticsearchSnapshotRepositoryUpdate,
		Delete: resourceElasticsearchSnapshotRepositoryDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"settings": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
		},
	}
}

func resourceElasticsearchSnapshotRepositoryCreate(d *schema.ResourceData, meta interface{}) error {
	err := resourceElasticsearchSnapshotRepositoryUpdate(d, meta)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))
	return nil
}

func resourceElasticsearchSnapshotRepositoryRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*elastic.Client)
	id := d.Id()
	repos, err := client.SnapshotGetRepository(id).Do(context.TODO())
	if err != nil {
		return err
	}
	d.Set("name", id)
	d.Set("type", repos[id].Type)
	d.Set("settings", repos[id].Settings)
	return nil
}

func resourceElasticsearchSnapshotRepositoryUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*elastic.Client)
	repo := elastic.SnapshotRepositoryMetaData{
		Type: d.Get("type").(string),
	}
	if v, ok := d.GetOk("settings"); ok {
		repo.Settings = v.(map[string]interface{})
	}

	_, err := client.SnapshotCreateRepository(d.Get("name").(string)).BodyJson(&repo).Do(context.TODO())
	return err
}

func resourceElasticsearchSnapshotRepositoryDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*elastic.Client)
	_, err := client.SnapshotDeleteRepository(d.Id()).Do(context.TODO())
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
