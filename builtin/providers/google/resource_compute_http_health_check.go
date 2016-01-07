package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeHttpHealthCheck() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeHttpHealthCheckCreate,
		Read:   resourceComputeHttpHealthCheckRead,
		Delete: resourceComputeHttpHealthCheckDelete,
		Update: resourceComputeHttpHealthCheckUpdate,

		Schema: map[string]*schema.Schema{
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

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  80,
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
		config.Project, hchk).Do()
	if err != nil {
		return fmt.Errorf("Error creating HttpHealthCheck: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(hchk.Name)

	err = computeOperationWaitGlobal(config, op, "Creating Http Health Check")
	if err != nil {
		return err
	}

	return resourceComputeHttpHealthCheckRead(d, meta)
}

func resourceComputeHttpHealthCheckUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

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
		config.Project, hchk.Name, hchk).Do()
	if err != nil {
		return fmt.Errorf("Error patching HttpHealthCheck: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(hchk.Name)

	err = computeOperationWaitGlobal(config, op, "Updating Http Health Check")
	if err != nil {
		return err
	}

	return resourceComputeHttpHealthCheckRead(d, meta)
}

func resourceComputeHttpHealthCheckRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	hchk, err := config.clientCompute.HttpHealthChecks.Get(
		config.Project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			log.Printf("[WARN] Removing HTTP Health Check %q because it's gone", d.Get("name").(string))
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading HttpHealthCheck: %s", err)
	}

	d.Set("host", hchk.Host)
	d.Set("request_path", hchk.RequestPath)
	d.Set("check_interval_sec", hchk.CheckIntervalSec)
	d.Set("health_threshold", hchk.HealthyThreshold)
	d.Set("port", hchk.Port)
	d.Set("timeout_sec", hchk.TimeoutSec)
	d.Set("unhealthy_threshold", hchk.UnhealthyThreshold)
	d.Set("self_link", hchk.SelfLink)

	return nil
}

func resourceComputeHttpHealthCheckDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Delete the HttpHealthCheck
	op, err := config.clientCompute.HttpHealthChecks.Delete(
		config.Project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting HttpHealthCheck: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, "Deleting Http Health Check")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
