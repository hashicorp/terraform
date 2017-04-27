package heroku

import (
	"context"
	"log"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceHerokuAppFeature() *schema.Resource {
	return &schema.Resource{
		Create: resourceAppFeatureCreate,
		Update: resourceAppFeatureUpdate,
		Read:   resourceAppFeatureRead,
		Delete: resourceAppFeatureDelete,

		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceAppFeatureRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	app, id := parseCompositeID(d.Id())

	feature, err := client.AppFeatureInfo(context.TODO(), app, id)
	if err != nil {
		return err
	}

	d.Set("app", app)
	d.Set("name", feature.Name)
	d.Set("enabled", feature.Enabled)

	return nil
}

func resourceAppFeatureCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	app := d.Get("app").(string)
	featureName := d.Get("name").(string)
	enabled := d.Get("enabled").(bool)

	opts := heroku.AppFeatureUpdateOpts{Enabled: enabled}

	log.Printf("[DEBUG] Feature set configuration: %#v, %#v", featureName, opts)

	feature, err := client.AppFeatureUpdate(context.TODO(), app, featureName, opts)
	if err != nil {
		return err
	}

	d.SetId(buildCompositeID(app, feature.ID))

	return resourceAppFeatureRead(d, meta)
}

func resourceAppFeatureUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("enabled") {
		return resourceAppFeatureCreate(d, meta)
	}

	return resourceAppFeatureRead(d, meta)
}

func resourceAppFeatureDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	app, id := parseCompositeID(d.Id())
	featureName := d.Get("name").(string)

	log.Printf("[INFO] Deleting app feature %s (%s) for app %s", featureName, id, app)
	opts := heroku.AppFeatureUpdateOpts{Enabled: false}
	_, err := client.AppFeatureUpdate(context.TODO(), app, id, opts)
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
