package heroku

import (
	"context"
	"fmt"
	"log"
	"time"

	heroku "github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceHerokuSpace() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuSpaceCreate,
		Read:   resourceHerokuSpaceRead,
		Update: resourceHerokuSpaceUpdate,
		Delete: resourceHerokuSpaceDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"organization": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"region": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceHerokuSpaceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	opts := heroku.SpaceCreateOpts{}
	opts.Name = d.Get("name").(string)
	opts.Organization = d.Get("organization").(string)

	if v, ok := d.GetOk("region"); ok {
		vs := v.(string)
		opts.Region = &vs
	}

	space, err := client.SpaceCreate(context.TODO(), opts)
	if err != nil {
		return err
	}

	d.SetId(space.ID)
	log.Printf("[INFO] Space ID: %s", d.Id())

	// Wait for the Space to be allocated
	log.Printf("[DEBUG] Waiting for Space (%s) to be allocated", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending: []string{"allocating"},
		Target:  []string{"allocated"},
		Refresh: SpaceStateRefreshFunc(client, d.Id()),
		Timeout: 20 * time.Minute,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Space (%s) to become available: %s", d.Id(), err)
	}

	return resourceHerokuSpaceRead(d, meta)
}

func resourceHerokuSpaceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	spaceRaw, _, err := SpaceStateRefreshFunc(client, d.Id())()
	if err != nil {
		return err
	}
	space := spaceRaw.(*heroku.Space)

	setSpaceAttributes(d, space)
	return nil
}

func resourceHerokuSpaceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	if !d.HasChange("name") {
		return nil
	}

	name := d.Get("name").(string)
	opts := heroku.SpaceUpdateOpts{Name: &name}

	space, err := client.SpaceUpdate(context.TODO(), d.Id(), opts)
	if err != nil {
		return err
	}

	// The type conversion here can be dropped when the vendored version of
	// heroku-go is updated.
	setSpaceAttributes(d, (*heroku.Space)(space))
	return nil
}

func setSpaceAttributes(d *schema.ResourceData, space *heroku.Space) {
	d.Set("name", space.Name)
	d.Set("organization", space.Organization.Name)
	d.Set("region", space.Region.Name)
}

func resourceHerokuSpaceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	log.Printf("[INFO] Deleting space: %s", d.Id())
	_, err := client.SpaceDelete(context.TODO(), d.Id())
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

// SpaceStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// a Space.
func SpaceStateRefreshFunc(client *heroku.Service, id string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		space, err := client.SpaceInfo(context.TODO(), id)
		if err != nil {
			return nil, "", err
		}

		// The type conversion here can be dropped when the vendored version of
		// heroku-go is updated.
		return (*heroku.Space)(space), space.State, nil
	}
}
