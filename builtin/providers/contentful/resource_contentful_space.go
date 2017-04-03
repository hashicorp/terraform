package contentful

import (
	contentful "github.com/contentful-labs/contentful-go"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceContentfulSpace() *schema.Resource {
	return &schema.Resource{
		Create: resourceSpaceCreate,
		Read:   resourceSpaceRead,
		Update: resourceSpaceUpdate,
		Delete: resourceSpaceDelete,

		Schema: map[string]*schema.Schema{
			"id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"version": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			// Space specific props
			"default_locale": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "en",
			},
		},
	}
}

func resourceSpaceCreate(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Contentful)

	space := &contentful.Space{
		Name:          d.Get("name").(string),
		DefaultLocale: d.Get("default_locale").(string),
	}

	err = client.Spaces.Upsert(space)
	if err != nil {
		return err
	}

	err = updateSpaceProperties(d, space)
	if err != nil {
		return err
	}

	d.SetId(space.Sys.ID)

	return nil
}

func resourceSpaceRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*contentful.Contentful)
	spaceID := d.Id()

	_, err := client.Spaces.Get(spaceID)
	if _, ok := err.(contentful.NotFoundError); ok {
		d.SetId("")
		return nil
	}

	return err
}

func resourceSpaceUpdate(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Contentful)
	spaceID := d.Id()

	space, err := client.Spaces.Get(spaceID)
	if err != nil {
		return err
	}

	space.Name = d.Get("name").(string)

	err = client.Spaces.Upsert(space)
	if err != nil {
		return err
	}

	return updateSpaceProperties(d, space)
}

func resourceSpaceDelete(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*contentful.Contentful)
	spaceID := d.Id()

	space, err := client.Spaces.Get(spaceID)
	if err != nil {
		return err
	}

	err = client.Spaces.Delete(space)
	if _, ok := err.(contentful.NotFoundError); ok {
		return nil
	}

	return err
}

func updateSpaceProperties(d *schema.ResourceData, space *contentful.Space) error {
	err := d.Set("version", space.Sys.Version)
	if err != nil {
		return err
	}

	err = d.Set("name", space.Name)
	if err != nil {
		return err
	}

	return nil
}
