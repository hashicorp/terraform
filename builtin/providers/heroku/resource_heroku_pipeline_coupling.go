package heroku

import (
	"context"
	"fmt"
	"log"

	"github.com/cyberdelia/heroku-go/v3"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceHerokuPipelineCoupling() *schema.Resource {
	return &schema.Resource{
		Create: resourceHerokuPipelineCouplingCreate,
		Read:   resourceHerokuPipelineCouplingRead,
		Delete: resourceHerokuPipelineCouplingDelete,

		Schema: map[string]*schema.Schema{
			"app": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"pipeline": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateUUID,
			},
			"stage": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validatePipelineStageName,
			},
		},
	}
}

func resourceHerokuPipelineCouplingCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	opts := heroku.PipelineCouplingCreateOpts{
		App:      d.Get("app").(string),
		Pipeline: d.Get("pipeline").(string),
		Stage:    d.Get("stage").(string),
	}

	log.Printf("[DEBUG] PipelineCoupling create configuration: %#v", opts)

	p, err := client.PipelineCouplingCreate(context.TODO(), opts)
	if err != nil {
		return fmt.Errorf("Error creating pipeline: %s", err)
	}

	d.SetId(p.ID)

	log.Printf("[INFO] PipelineCoupling ID: %s", d.Id())

	return resourceHerokuPipelineCouplingRead(d, meta)
}

func resourceHerokuPipelineCouplingDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	log.Printf("[INFO] Deleting pipeline: %s", d.Id())

	_, err := client.PipelineCouplingDelete(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("Error deleting pipeline: %s", err)
	}

	return nil
}

func resourceHerokuPipelineCouplingRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*heroku.Service)

	p, err := client.PipelineCouplingInfo(context.TODO(), d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving pipeline: %s", err)
	}

	d.Set("app", p.App)
	d.Set("pipeline", p.Pipeline)
	d.Set("stage", p.Stage)

	return nil
}
