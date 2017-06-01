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

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"app_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
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

	// grab App info
	app, err := client.AppInfo(context.TODO(), p.App.ID)
	if err != nil {
		log.Printf("[WARN] Error looking up addional App info for pipeline coupling (%s): %s", d.Id(), err)
	} else {
		d.Set("app", app.Name)
	}

	d.Set("app_id", p.App.ID)
	d.Set("stage", p.Stage)
	d.Set("pipeline", p.Pipeline.ID)

	return nil
}
