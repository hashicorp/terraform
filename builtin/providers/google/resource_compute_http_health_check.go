package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
)

func resourceComputeHttpHealthCheck() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeHttpHealthCheckCreate,
		Read:   resourceComputeHttpHealthCheckRead,
		Delete: resourceComputeHttpHealthCheckDelete,
		Update: resourceComputeHttpHealthCheckUpdate,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"check_interval_sec": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  5,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"healthy_threshold": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  2,
			},

			"host": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  80,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"request_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "/",
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"timeout_sec": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  5,
			},

			"unhealthy_threshold": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  2,
			},
		},
	}
}

func resourceComputeHttpHealthCheckCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Build the parameter
	hchk := &compute.HttpHealthCheck{
		Name: d.Get("name").(string),
	}
	// Optional things
	if v, ok := d.GetOk("description"); ok {
		hchk.Description = v.(string)
	}
	if v, ok := d.GetOk("host"); ok {
		hchk.Host = v.(string)
	}
	if v, ok := d.GetOk("request_path"); ok {
		hchk.RequestPath = v.(string)
	}
	if v, ok := d.GetOk("check_interval_sec"); ok {
		hchk.CheckIntervalSec = int64(v.(int))
	}
	if v, ok := d.GetOk("healthy_threshold"); ok {
		hchk.HealthyThreshold = int64(v.(int))
	}
	if v, ok := d.GetOk("port"); ok {
		hchk.Port = int64(v.(int))
	}
	if v, ok := d.GetOk("timeout_sec"); ok {
		hchk.TimeoutSec = int64(v.(int))
	}
	if v, ok := d.GetOk("unhealthy_threshold"); ok {
		hchk.UnhealthyThreshold = int64(v.(int))
	}

	log.Printf("[DEBUG] HttpHealthCheck insert request: %#v", hchk)
	op, err := config.clientCompute.HttpHealthChecks.Insert(
		project, hchk).Do()
	if err != nil {
		return fmt.Errorf("Error creating HttpHealthCheck: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(hchk.Name)

	err = computeOperationWaitGlobal(config, op, project, "Creating Http Health Check")
	if err != nil {
		return err
	}

	return resourceComputeHttpHealthCheckRead(d, meta)
}

func resourceComputeHttpHealthCheckUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Build the parameter
	hchk := &compute.HttpHealthCheck{
		Name: d.Get("name").(string),
	}
	// Optional things
	if v, ok := d.GetOk("description"); ok {
		hchk.Description = v.(string)
	}
	if v, ok := d.GetOk("host"); ok {
		hchk.Host = v.(string)
	}
	if v, ok := d.GetOk("request_path"); ok {
		hchk.RequestPath = v.(string)
	}
	if v, ok := d.GetOk("check_interval_sec"); ok {
		hchk.CheckIntervalSec = int64(v.(int))
	}
	if v, ok := d.GetOk("healthy_threshold"); ok {
		hchk.HealthyThreshold = int64(v.(int))
	}
	if v, ok := d.GetOk("port"); ok {
		hchk.Port = int64(v.(int))
	}
	if v, ok := d.GetOk("timeout_sec"); ok {
		hchk.TimeoutSec = int64(v.(int))
	}
	if v, ok := d.GetOk("unhealthy_threshold"); ok {
		hchk.UnhealthyThreshold = int64(v.(int))
	}

	log.Printf("[DEBUG] HttpHealthCheck patch request: %#v", hchk)
	op, err := config.clientCompute.HttpHealthChecks.Patch(
		project, hchk.Name, hchk).Do()
	if err != nil {
		return fmt.Errorf("Error patching HttpHealthCheck: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(hchk.Name)

	err = computeOperationWaitGlobal(config, op, project, "Updating Http Health Check")
	if err != nil {
		return err
	}

	return resourceComputeHttpHealthCheckRead(d, meta)
}

func resourceComputeHttpHealthCheckRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	hchk, err := config.clientCompute.HttpHealthChecks.Get(
		project, d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("HTTP Health Check %q", d.Get("name").(string)))
	}

	d.Set("host", hchk.Host)
	d.Set("request_path", hchk.RequestPath)
	d.Set("check_interval_sec", hchk.CheckIntervalSec)
	d.Set("healthy_threshold", hchk.HealthyThreshold)
	d.Set("port", hchk.Port)
	d.Set("timeout_sec", hchk.TimeoutSec)
	d.Set("unhealthy_threshold", hchk.UnhealthyThreshold)
	d.Set("self_link", hchk.SelfLink)
	d.Set("name", hchk.Name)
	d.Set("description", hchk.Description)
	d.Set("project", project)

	return nil
}

func resourceComputeHttpHealthCheckDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Delete the HttpHealthCheck
	op, err := config.clientCompute.HttpHealthChecks.Delete(
		project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting HttpHealthCheck: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Deleting Http Health Check")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
