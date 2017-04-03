package contentful

import (
	"github.com/hashicorp/terraform/helper/schema"
	contentful "github.com/tolgaakyuz/contentful-go"
)

func resourceContentfulAPIKey() *schema.Resource {
	return &schema.Resource{
		Create: resourceCreateAPIKey,
		Read:   resourceReadAPIKey,
		Update: resourceUpdateAPIKey,
		Delete: resourceDeleteAPIKey,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"version": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"space_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

func resourceCreateAPIKey(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Contentful)

	apiKey := &contentful.APIKey{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
	}

	err = client.APIKeys.Upsert(d.Get("space_id").(string), apiKey)
	if err != nil {
		return err
	}

	if err := setAPIKeyProperties(d, apiKey); err != nil {
		return err
	}

	d.SetId(apiKey.Sys.ID)

	return nil
}

func resourceUpdateAPIKey(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Contentful)
	spaceID := d.Get("space_id").(string)
	apiKeyID := d.Id()

	apiKey, err := client.APIKeys.Get(spaceID, apiKeyID)
	if err != nil {
		return err
	}

	apiKey.Name = d.Get("name").(string)
	apiKey.Description = d.Get("description").(string)

	err = client.APIKeys.Upsert(spaceID, apiKey)
	if err != nil {
		return err
	}

	if err := setAPIKeyProperties(d, apiKey); err != nil {
		return err
	}

	d.SetId(apiKey.Sys.ID)

	return nil
}

func resourceReadAPIKey(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Contentful)
	spaceID := d.Get("space_id").(string)
	apiKeyID := d.Id()

	apiKey, err := client.APIKeys.Get(spaceID, apiKeyID)
	if _, ok := err.(contentful.NotFoundError); ok {
		d.SetId("")
		return nil
	}

	return setAPIKeyProperties(d, apiKey)
}

func resourceDeleteAPIKey(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Contentful)
	spaceID := d.Get("space_id").(string)
	apiKeyID := d.Id()

	apiKey, err := client.APIKeys.Get(spaceID, apiKeyID)
	if err != nil {
		return err
	}

	return client.APIKeys.Delete(spaceID, apiKey)
}

func setAPIKeyProperties(d *schema.ResourceData, apiKey *contentful.APIKey) error {
	if err := d.Set("space_id", apiKey.Sys.Space.Sys.ID); err != nil {
		return err
	}

	if err := d.Set("version", apiKey.Sys.Version); err != nil {
		return err
	}

	if err := d.Set("name", apiKey.Name); err != nil {
		return err
	}

	if err := d.Set("description", apiKey.Description); err != nil {
		return err
	}

	return nil
}
