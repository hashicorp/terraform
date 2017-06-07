package google

import (
	"bytes"
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"google.golang.org/api/compute/v1"
)

func resourceComputeBackendService() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeBackendServiceCreate,
		Read:   resourceComputeBackendServiceRead,
		Update: resourceComputeBackendServiceUpdate,
		Delete: resourceComputeBackendServiceDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					value := v.(string)
					re := `^(?:[a-z](?:[-a-z0-9]{0,61}[a-z0-9])?)$`
					if !regexp.MustCompile(re).MatchString(value) {
						errors = append(errors, fmt.Errorf(
							"%q (%q) doesn't match regexp %q", k, value, re))
					}
					return
				},
			},

			"health_checks": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Required: true,
				Set:      schema.HashString,
			},

			"backend": &schema.Schema{
				Type: schema.TypeSet,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"group": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"balancing_mode": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Default:  "UTILIZATION",
						},
						"capacity_scaler": &schema.Schema{
							Type:     schema.TypeFloat,
							Optional: true,
							Default:  1,
						},
						"description": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"max_rate": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"max_rate_per_instance": &schema.Schema{
							Type:     schema.TypeFloat,
							Optional: true,
						},
						"max_utilization": &schema.Schema{
							Type:     schema.TypeFloat,
							Optional: true,
							Default:  0.8,
						},
					},
				},
				Optional: true,
				Set:      resourceGoogleComputeBackendServiceBackendHash,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"enable_cdn": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"fingerprint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"port_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Removed:  "region has been removed as it was never used. For internal load balancing, use google_compute_region_backend_service",
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"session_affinity": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"timeout_sec": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"connection_draining_timeout_sec": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
		},
	}
}

func resourceComputeBackendServiceCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	hc := d.Get("health_checks").(*schema.Set).List()
	healthChecks := make([]string, 0, len(hc))
	for _, v := range hc {
		healthChecks = append(healthChecks, v.(string))
	}

	service := compute.BackendService{
		Name:         d.Get("name").(string),
		HealthChecks: healthChecks,
	}

	if v, ok := d.GetOk("backend"); ok {
		service.Backends = expandBackends(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("description"); ok {
		service.Description = v.(string)
	}

	if v, ok := d.GetOk("port_name"); ok {
		service.PortName = v.(string)
	}

	if v, ok := d.GetOk("protocol"); ok {
		service.Protocol = v.(string)
	}

	if v, ok := d.GetOk("session_affinity"); ok {
		service.SessionAffinity = v.(string)
	}

	if v, ok := d.GetOk("timeout_sec"); ok {
		service.TimeoutSec = int64(v.(int))
	}

	if v, ok := d.GetOk("enable_cdn"); ok {
		service.EnableCDN = v.(bool)
	}

	if v, ok := d.GetOk("connection_draining_timeout_sec"); ok {
		connectionDraining := &compute.ConnectionDraining{
			DrainingTimeoutSec: int64(v.(int)),
		}

		service.ConnectionDraining = connectionDraining
	}

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating new Backend Service: %#v", service)
	op, err := config.clientCompute.BackendServices.Insert(
		project, &service).Do()
	if err != nil {
		return fmt.Errorf("Error creating backend service: %s", err)
	}

	log.Printf("[DEBUG] Waiting for new backend service, operation: %#v", op)

	// Store the ID now
	d.SetId(service.Name)

	// Wait for the operation to complete
	waitErr := computeOperationWaitGlobal(config, op, project, "Creating Backend Service")
	if waitErr != nil {
		// The resource didn't actually create
		d.SetId("")
		return waitErr
	}

	return resourceComputeBackendServiceRead(d, meta)
}

func resourceComputeBackendServiceRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	service, err := config.clientCompute.BackendServices.Get(
		project, d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Backend Service %q", d.Get("name").(string)))
	}

	d.Set("description", service.Description)
	d.Set("enable_cdn", service.EnableCDN)
	d.Set("port_name", service.PortName)
	d.Set("protocol", service.Protocol)
	d.Set("session_affinity", service.SessionAffinity)
	d.Set("timeout_sec", service.TimeoutSec)
	d.Set("fingerprint", service.Fingerprint)
	d.Set("self_link", service.SelfLink)
	d.Set("backend", flattenBackends(service.Backends))
	d.Set("connection_draining_timeout_sec", service.ConnectionDraining.DrainingTimeoutSec)

	d.Set("health_checks", service.HealthChecks)

	return nil
}

func resourceComputeBackendServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	hc := d.Get("health_checks").(*schema.Set).List()
	healthChecks := make([]string, 0, len(hc))
	for _, v := range hc {
		healthChecks = append(healthChecks, v.(string))
	}

	service := compute.BackendService{
		Name:         d.Get("name").(string),
		Fingerprint:  d.Get("fingerprint").(string),
		HealthChecks: healthChecks,
	}

	// Optional things
	if v, ok := d.GetOk("backend"); ok {
		service.Backends = expandBackends(v.(*schema.Set).List())
	}
	if v, ok := d.GetOk("description"); ok {
		service.Description = v.(string)
	}
	if v, ok := d.GetOk("port_name"); ok {
		service.PortName = v.(string)
	}
	if v, ok := d.GetOk("protocol"); ok {
		service.Protocol = v.(string)
	}
	if v, ok := d.GetOk("timeout_sec"); ok {
		service.TimeoutSec = int64(v.(int))
	}

	if d.HasChange("connection_draining_timeout_sec") {
		connectionDraining := &compute.ConnectionDraining{
			DrainingTimeoutSec: int64(d.Get("connection_draining_timeout_sec").(int)),
		}

		service.ConnectionDraining = connectionDraining
	}

	if d.HasChange("session_affinity") {
		service.SessionAffinity = d.Get("session_affinity").(string)
	}

	if d.HasChange("enable_cdn") {
		service.EnableCDN = d.Get("enable_cdn").(bool)
	}

	log.Printf("[DEBUG] Updating existing Backend Service %q: %#v", d.Id(), service)
	op, err := config.clientCompute.BackendServices.Update(
		project, d.Id(), &service).Do()
	if err != nil {
		return fmt.Errorf("Error updating backend service: %s", err)
	}

	d.SetId(service.Name)

	err = computeOperationWaitGlobal(config, op, project, "Updating Backend Service")
	if err != nil {
		return err
	}

	return resourceComputeBackendServiceRead(d, meta)
}

func resourceComputeBackendServiceDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Deleting backend service %s", d.Id())
	op, err := config.clientCompute.BackendServices.Delete(
		project, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting backend service: %s", err)
	}

	err = computeOperationWaitGlobal(config, op, project, "Deleting Backend Service")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func expandBackends(configured []interface{}) []*compute.Backend {
	backends := make([]*compute.Backend, 0, len(configured))

	for _, raw := range configured {
		data := raw.(map[string]interface{})

		b := compute.Backend{
			Group: data["group"].(string),
		}

		if v, ok := data["balancing_mode"]; ok {
			b.BalancingMode = v.(string)
		}
		if v, ok := data["capacity_scaler"]; ok {
			b.CapacityScaler = v.(float64)
		}
		if v, ok := data["description"]; ok {
			b.Description = v.(string)
		}
		if v, ok := data["max_rate"]; ok {
			b.MaxRate = int64(v.(int))
		}
		if v, ok := data["max_rate_per_instance"]; ok {
			b.MaxRatePerInstance = v.(float64)
		}
		if v, ok := data["max_utilization"]; ok {
			b.MaxUtilization = v.(float64)
		}

		backends = append(backends, &b)
	}

	return backends
}

func flattenBackends(backends []*compute.Backend) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(backends))

	for _, b := range backends {
		data := make(map[string]interface{})

		data["balancing_mode"] = b.BalancingMode
		data["capacity_scaler"] = b.CapacityScaler
		data["description"] = b.Description
		data["group"] = b.Group
		data["max_rate"] = b.MaxRate
		data["max_rate_per_instance"] = b.MaxRatePerInstance
		data["max_utilization"] = b.MaxUtilization

		result = append(result, data)
	}

	return result
}

func resourceGoogleComputeBackendServiceBackendHash(v interface{}) int {
	if v == nil {
		return 0
	}

	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["group"].(string)))

	if v, ok := m["balancing_mode"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["capacity_scaler"]; ok {
		buf.WriteString(fmt.Sprintf("%f-", v.(float64)))
	}
	if v, ok := m["description"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["max_rate"]; ok {
		buf.WriteString(fmt.Sprintf("%d-", int64(v.(int))))
	}
	if v, ok := m["max_rate_per_instance"]; ok {
		buf.WriteString(fmt.Sprintf("%f-", v.(float64)))
	}
	if v, ok := m["max_rate_per_instance"]; ok {
		buf.WriteString(fmt.Sprintf("%f-", v.(float64)))
	}

	return hashcode.String(buf.String())
}
