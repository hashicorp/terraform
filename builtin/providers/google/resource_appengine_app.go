package google

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"google.golang.org/api/appengine/v1"
	"google.golang.org/api/googleapi"
)

func resourceAppengineApp() *schema.Resource {
	return &schema.Resource{
		Create: resourceAppengineAppCreate,
		Read:   resourceAppengineAppRead,
		Delete: resourceAppengineAppDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAppengineAppCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	region := d.Get("region").(string)

	app := &appengine.Application{Id: project, LocationId: region}

	var res *appengine.Operation

	err = resource.Retry(1*time.Minute, func() *resource.RetryError {
		call := config.clientAppengine.Apps.Create(app)

		res, err = call.Do()
		if err == nil {
			return nil
		}
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 429 {
			return resource.RetryableError(gerr)
		}
		return resource.NonRetryableError(err)
	})

	if err != nil {
		fmt.Printf("Error creating app engine app %s: %v", project, err)
		return err
	}

	log.Printf("[DEBUG] Created App Engine app %v at location %v\n\n", project, region)

	d.SetId(project)

	return nil
}

func resourceAppengineAppRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	res, err := config.clientAppengine.Apps.Get(project).Do()

	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing App Engine app %q because it's gone", project)
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading App Engine app %s: %v", project, err)
	}

	log.Printf("[DEBUG] Read App Engine app %v at location %v\n\n", res.Id, res.LocationId)

	d.SetId(res.Id)

	return nil
}

func resourceAppengineAppDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Deleting App Engine app not supported %v\n\n", project)

	return nil
}
