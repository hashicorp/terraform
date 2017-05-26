package heroku

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceHerokuDrain() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuDrainCreate,
		Read:   resourceHerokuDrainRead,
		Delete: resourceHerokuDrainDelete,

		Schema: map[string]*schema.Schema{
			"url": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"token": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

const retryableError = `App hasn't yet been assigned a log channel. Please try again momentarily.`

func resourceHerokuDrainCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	app := d.Get("app").(string)
	url := d.Get("url").(string)

	log.Printf("[DEBUG] Drain create configuration: %#v, %#v", app, url)

	var dr *heroku.LogDrain
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		d, err := client.LogDrainCreate(context.TODO(), app, heroku.LogDrainCreateOpts{URL: url})
		if err != nil {
			if strings.Contains(err.Error(), retryableError) {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		dr = d
		return nil
	})
	if err != nil {
		return err
	}

	d.SetId(dr.ID)
	d.Set("url", dr.URL)
	d.Set("token", dr.Token)

	log.Printf("[INFO] Drain ID: %s", d.Id())
	return nil
}

func resourceHerokuDrainDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	log.Printf("[INFO] Deleting drain: %s", d.Id())

	// Destroy the drain
	_, err := client.LogDrainDelete(context.TODO(), d.Get("app").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting drain: %s", err)
	}

	return nil
}

func resourceHerokuDrainRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	dr, err := client.LogDrainInfo(context.TODO(), d.Get("app").(string), d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving drain: %s", err)
	}

	d.Set("url", dr.URL)
	d.Set("token", dr.Token)

	return nil
}
