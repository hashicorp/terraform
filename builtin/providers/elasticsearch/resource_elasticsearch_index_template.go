package elasticsearch

import (
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform/helper/schema"
	elastic "gopkg.in/olivere/elastic.v5"
)

func resourceElasticsearchIndexTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceElasticsearchIndexTemplateCreate,
		Read:   resourceElasticsearchIndexTemplateRead,
		Update: resourceElasticsearchIndexTemplateUpdate,
		Delete: resourceElasticsearchIndexTemplateDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"body": &schema.Schema{
				Type:             schema.TypeString,
				Required:         true,
				DiffSuppressFunc: diffSuppressIndexTemplate,
			},
		},
	}
}

func resourceElasticsearchIndexTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	err := resourceElasticsearchPutIndexTemplate(d, meta, true)
	if err != nil {
		return err
	}
	d.SetId(d.Get("name").(string))
	return nil
}

func resourceElasticsearchIndexTemplateRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*elastic.Client)
	res, err := client.IndexGetTemplate(d.Id()).Do(context.TODO())
	if err != nil {
		return err
	}
	t := res[d.Id()]
	tj, err := json.Marshal(t)
	if err != nil {
		return err
	}
	d.Set("name", d.Id())
	d.Set("body", string(tj))
	return nil
}

func resourceElasticsearchIndexTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	return resourceElasticsearchPutIndexTemplate(d, meta, false)
}

func resourceElasticsearchIndexTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*elastic.Client)
	_, err := client.IndexDeleteTemplate(d.Id()).Do(context.TODO())
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}

func resourceElasticsearchPutIndexTemplate(d *schema.ResourceData, meta interface{}, create bool) error {
	client := meta.(*elastic.Client)
	name := d.Get("name").(string)
	body := d.Get("body").(string)
	_, err := client.IndexPutTemplate(name).BodyString(body).Create(create).Do(context.TODO())
	return err
}
