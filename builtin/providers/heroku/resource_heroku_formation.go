package heroku

import (
	"fmt"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceHerokuFormation() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuFormationCreate,
		Read:   resourceHerokuFormationRead,
		Update: resourceHerokuFormationUpdate,
		Delete: resourceHerokuFormationDelete,
		Exists: resourceHerokuFormationExists,

		Schema: map[string]*schema.Schema{
			"app": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"quantity": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"size": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}
func resourceHerokuFormationCreate(d *schema.ResourceData, meta interface{}) error {

	d.SetId("")
	return resourceHerokuFormationUpdate(d, meta)
}

func resourceHerokuFormationRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	formations, err := client.FormationList(d.Get("app").(string), (&heroku.ListRange{Max: 10000}))

	if err != nil {
		return fmt.Errorf("Error retrieving formation list: %s", err)
	}

	for _, formation := range formations {
		if formation.Type == d.Get("type").(string) {
			d.Set("size", formation.Size)
			d.Set("quantity", formation.Quantity)
			d.SetId(formation.ID)
		}
	}

	return nil
}

func resourceHerokuFormationUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	app := d.Get("app").(string)

	quantity := d.Get("quantity").(int)
	size := d.Get("size").(string)

	if d.Id() == d.Get("type") && (d.HasChange("quantity") || d.HasChange("size")) {
		opts := heroku.FormationUpdateOpts{
			Quantity: &quantity,
		}

		if size != "" {
			opts.Size = &size
		}
		_, err := client.FormationUpdate(app, d.Get("type").(string), opts)
		if err != nil {
			return err
		}
	}

	return resourceHerokuFormationRead(d, meta)
}

func resourceHerokuFormationDelete(d *schema.ResourceData, meta interface{}) error {
	d.SetId("")
	return nil
}

func resourceHerokuFormationExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*heroku.Service)

	formations, err := client.FormationList(d.Get("app").(string), (&heroku.ListRange{Max: 10000}))

	if err != nil {
		return false, fmt.Errorf("Error retrieving formation list: %s", err)
	}

  exists := false
	for _, formation := range formations {
		if formation.Type == d.Get("type").(string) {
      exists = true
		}
	}

	return exists, nil
}
