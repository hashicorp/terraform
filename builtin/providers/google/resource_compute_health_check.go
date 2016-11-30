package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeHealthCheck() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeHealthCheckCreate,
		Read:   resourceComputeHealthCheckRead,
		Delete: resourceComputeHealthCheckDelete,
		Update: resourceComputeHealthCheckUpdate,
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

			"type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "TCP",
				ForceNew: true,
			},

			"tcp_health_check": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Default:  80,
						},
						"port_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"proxy_header": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "NONE",
						},
						"request": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"response": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
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

func resourceComputeHealthCheckCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Build the parameter
	hchk := &compute.HealthCheck{
		Name: d.Get("name").(string),
	}
	// Optional things
	if v, ok := d.GetOk("description"); ok {
		hchk.Description = v.(string)
	}
	if v, ok := d.GetOk("check_interval_sec"); ok {
		hchk.CheckIntervalSec = int64(v.(int))
	}
	if v, ok := d.GetOk("healthy_threshold"); ok {
		hchk.HealthyThreshold = int64(v.(int))
	}
	if v, ok := d.GetOk("timeout_sec"); ok {
		hchk.TimeoutSec = int64(v.(int))
	}
	if v, ok := d.GetOk("unhealthy_threshold"); ok {
		hchk.UnhealthyThreshold = int64(v.(int))
	}
	if v, ok := d.GetOk("type"); ok {
		hchk.Type = v.(string)
	}
	if v, ok := d.GetOk("tcp_health_check"); ok {
		// check that type is tcp?
		tcpcheck := v.([]interface{})[0].(map[string]interface{})
		tcpHealthCheck := &compute.TCPHealthCheck{}
		if val, ok := tcpcheck["port"]; ok {
			tcpHealthCheck.Port = int64(val.(int))
		}
		if val, ok := tcpcheck["port_name"]; ok {
			tcpHealthCheck.PortName = val.(string)
		}
		if val, ok := tcpcheck["proxy_header"]; ok {
			tcpHealthCheck.ProxyHeader = val.(string)
		}
		if val, ok := tcpcheck["request"]; ok {
			tcpHealthCheck.Request = val.(string)
		}
		if val, ok := tcpcheck["response"]; ok {
			tcpHealthCheck.Response = val.(string)
		}
		hchk.TcpHealthCheck = tcpHealthCheck
	}

	log.Printf("[DEBUG] HealthCheck insert request: %#v", hchk)
	op, err := config.clientCompute.HealthChecks.Insert(
		project, hchk).Do()
	if err != nil {
		return fmt.Errorf("Error creating HealthCheck: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(hchk.Name)

	err = computeOperationWaitGlobal(config, op, project, "Creating Health Check")
	if err != nil {
		return err
	}

	return resourceComputeHealthCheckRead(d, meta)
}

func resourceComputeHealthCheckUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Build the parameter
	hchk := &compute.HealthCheck{
		Name: d.Get("name").(string),
	}
	// Optional things
	if v, ok := d.GetOk("description"); ok {
		hchk.Description = v.(string)
	}
	if v, ok := d.GetOk("check_interval_sec"); ok {
		hchk.CheckIntervalSec = int64(v.(int))
	}
	if v, ok := d.GetOk("healthy_threshold"); ok {
		hchk.HealthyThreshold = int64(v.(int))
	}
	if v, ok := d.GetOk("timeout_sec"); ok {
		hchk.TimeoutSec = int64(v.(int))
	}
	if v, ok := d.GetOk("unhealthy_threshold"); ok {
		hchk.UnhealthyThreshold = int64(v.(int))
	}
	if v, ok := d.GetOk("type"); ok {
		hchk.Type = v.(string)
	}
	if v, ok := d.GetOk("tcp_health_check"); ok {
		// check that type is tcp?
		tcpcheck := v.([]interface{})[0].(map[string]interface{})
		var tcpHealthCheck *compute.TCPHealthCheck
		if val, ok := tcpcheck["port"]; ok {
			tcpHealthCheck.Port = int64(val.(int))
		}
		if val, ok := tcpcheck["port_name"]; ok {
			tcpHealthCheck.PortName = val.(string)
		}
		if val, ok := tcpcheck["proxy_header"]; ok {
			tcpHealthCheck.ProxyHeader = val.(string)
		}
		if val, ok := tcpcheck["request"]; ok {
			tcpHealthCheck.Request = val.(string)
		}
		if val, ok := tcpcheck["response"]; ok {
			tcpHealthCheck.Response = val.(string)
		}
		hchk.TcpHealthCheck = tcpHealthCheck
	}

	log.Printf("[DEBUG] HealthCheck patch request: %#v", hchk)
	op, err := config.clientCompute.HealthChecks.Patch(
		project, hchk.Name, hchk).Do()
	if err != nil {
		return fmt.Errorf("Error patching HealthCheck: %s", err)
	}

	// It probably maybe worked, so store the ID now
	d.SetId(hchk.Name)

	err = computeOperationWaitGlobal(config, op, project, "Updating Health Check")
	if err != nil {
		return err
	}

	return resourceComputeHealthCheckRead(d, meta)
}

func resourceComputeHealthCheckRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	hchk, err := config.clientCompute.HealthChecks.Get(
		project, d.Id()).Do()
	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			// The resource doesn't exist anymore
			log.Printf("[WARN] Removing Health Check %q because it's gone", d.Get("name").(string))
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error reading HealthCheck: %s", err)
	}

	d.Set("check_interval_sec", hchk.CheckIntervalSec)
	d.Set("healthy_threshold", hchk.HealthyThreshold)
	d.Set("timeout_sec", hchk.TimeoutSec)
	d.Set("unhealthy_threshold", hchk.UnhealthyThreshold)
	d.Set("type", hchk.Type)
	d.Set("tcp_health_check", hchk.TcpHealthCheck)
	d.Set("self_link", hchk.SelfLink)
	d.Set("name", hchk.Name)
	d.Set("description", hchk.Description)
	d.Set("project", project)

	return nil
}

func resourceComputeHealthCheckDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	// Delete the HealthCheck
	op, err := config.clientCompute.HealthChecks.Delete(
		project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting HealthCheck: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Deleting Health Check")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
