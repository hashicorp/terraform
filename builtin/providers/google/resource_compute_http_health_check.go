package google

import (
	"fmt"
	"log"
	"time"

	"code.google.com/p/google-api-go-client/compute/v1"
	"code.google.com/p/google-api-go-client/googleapi"
	"github.com/hashicorp/terraform/helper/schema"
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
				ForceNew: false,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"healthy_threshold": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
			},

			"host": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
			},

			"request_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"timeout_sec": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
			},

			"unhealthy_threshold": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: false,
			},
		},
	}
}

func resourceComputeHttpHealthCheckCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Build the parameter
	hchk := &compute.HttpHealthCheck{
		Description: d.Get("description").(string),
		Host: d.Get("host").(string),
		Name: d.Get("name").(string),
		RequestPath: d.Get("request_path").(string),
	}
	if d.Get("check_interval_sec") != nil {
		hchk.CheckIntervalSec = int64(d.Get("check_interval_sec").(int))
	}
	if d.Get("health_threshold") != nil {
		hchk.HealthyThreshold = int64(d.Get("healthy_threshold").(int))
	}
	if d.Get("port") != nil {
		hchk.Port = int64(d.Get("port").(int))
	}
	if d.Get("timeout") != nil {
		hchk.TimeoutSec = int64(d.Get("timeout_sec").(int))
	}
	if d.Get("unhealthy_threshold") != nil {
		hchk.UnhealthyThreshold = int64(d.Get("unhealthy_threshold").(int))
	}

	log.Printf("[DEBUG] HttpHealthCheck insert request: %#v", hchk)
	op, err := config.clientCompute.HttpHealthChecks.Insert(
		config.Project, hchk).Do()
	if err != nil {
		return fmt.Errorf("Error creating HttpHealthCheck: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(hchk.Name)

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Type:    OperationWaitGlobal,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for HttpHealthCheck to create: %s", err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// The resource didn't actually create
		d.SetId("")

		// Return the error
		return OperationError(*op.Error)
	}

	return resourceComputeHttpHealthCheckRead(d, meta)
}

func resourceComputeHttpHealthCheckUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	// Build the parameter
	hchk := &compute.HttpHealthCheck{
		Description: d.Get("description").(string),
		Host: d.Get("host").(string),
		Name: d.Get("name").(string),
		RequestPath: d.Get("request_path").(string),
	}
	if d.Get("check_interval_sec") != nil {
		hchk.CheckIntervalSec = int64(d.Get("check_interval_sec").(int))
	}
	if d.Get("health_threshold") != nil {
		hchk.HealthyThreshold = int64(d.Get("healthy_threshold").(int))
	}
	if d.Get("port") != nil {
		hchk.Port = int64(d.Get("port").(int))
	}
	if d.Get("timeout") != nil {
		hchk.TimeoutSec = int64(d.Get("timeout_sec").(int))
	}
	if d.Get("unhealthy_threshold") != nil {
		hchk.UnhealthyThreshold = int64(d.Get("unhealthy_threshold").(int))
	}

	log.Printf("[DEBUG] HttpHealthCheck patch request: %#v", hchk)
	op, err := config.clientCompute.HttpHealthChecks.Patch(
		config.Project, hchk.Name, hchk).Do()
	if err != nil {
		return fmt.Errorf("Error patching HttpHealthCheck: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(hchk.Name)

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Type:    OperationWaitGlobal,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for HttpHealthCheck to patch: %s", err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// The resource didn't actually create
		d.SetId("")

		// Return the error
		return OperationError(*op.Error)
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
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading HttpHealthCheck: %s", err)
	}

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

	// Wait for the operation to complete
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:      op,
		Project: config.Project,
		Type:    OperationWaitGlobal,
	}
	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for HttpHealthCheck to delete: %s", err)
	}
	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		// Return the error
		return OperationError(*op.Error)
	}

	d.SetId("")
	return nil
}
