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

func resourceComputeRegionBackendService() *schema.Resource {
	return &schema.Resource{
		Create: resourceComputeRegionBackendServiceCreate,
		Read:   resourceComputeRegionBackendServiceRead,
		Update: resourceComputeRegionBackendServiceUpdate,
		Delete: resourceComputeRegionBackendServiceDelete,

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
						"description": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Optional: true,
				Set:      resourceGoogleComputeRegionBackendServiceBackendHash,
			},

			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"fingerprint": &schema.Schema{
				Type:     schema.TypeString,
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

			"session_affinity": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"timeout_sec": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceComputeRegionBackendServiceCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	hc := d.Get("health_checks").(*schema.Set).List()
	healthChecks := make([]string, 0, len(hc))
	for _, v := range hc {
		healthChecks = append(healthChecks, v.(string))
	}

	service := compute.BackendService{
		Name:                d.Get("name").(string),
		HealthChecks:        healthChecks,
		LoadBalancingScheme: "INTERNAL",
	}

	if v, ok := d.GetOk("backend"); ok {
		service.Backends = expandBackends(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("description"); ok {
		service.Description = v.(string)
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

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Creating new Region Backend Service: %#v", service)

	op, err := config.clientCompute.RegionBackendServices.Insert(
		project, region, &service).Do()
	if err != nil {
		return fmt.Errorf("Error creating backend service: %s", err)
	}

	log.Printf("[DEBUG] Waiting for new backend service, operation: %#v", op)

	d.SetId(service.Name)

	err = computeOperationWaitRegion(config, op, project, region, "Creating Region Backend Service")
	if err != nil {
		return err
	}

	return resourceComputeRegionBackendServiceRead(d, meta)
}

func resourceComputeRegionBackendServiceRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	service, err := config.clientCompute.RegionBackendServices.Get(
		project, region, d.Id()).Do()
	if err != nil {
		return handleNotFoundError(err, d, fmt.Sprintf("Region Backend Service %q", d.Get("name").(string)))
	}

	d.Set("description", service.Description)
	d.Set("protocol", service.Protocol)
	d.Set("session_affinity", service.SessionAffinity)
	d.Set("timeout_sec", service.TimeoutSec)
	d.Set("fingerprint", service.Fingerprint)
	d.Set("self_link", service.SelfLink)

	d.Set("backend", flattenBackends(service.Backends))
	d.Set("health_checks", service.HealthChecks)

	return nil
}

func resourceComputeRegionBackendServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	hc := d.Get("health_checks").(*schema.Set).List()
	healthChecks := make([]string, 0, len(hc))
	for _, v := range hc {
		healthChecks = append(healthChecks, v.(string))
	}

	service := compute.BackendService{
		Name:                d.Get("name").(string),
		Fingerprint:         d.Get("fingerprint").(string),
		HealthChecks:        healthChecks,
		LoadBalancingScheme: "INTERNAL",
	}

	// Optional things
	if v, ok := d.GetOk("backend"); ok {
		service.Backends = expandBackends(v.(*schema.Set).List())
	}
	if v, ok := d.GetOk("description"); ok {
		service.Description = v.(string)
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

	log.Printf("[DEBUG] Updating existing Backend Service %q: %#v", d.Id(), service)
	op, err := config.clientCompute.RegionBackendServices.Update(
		project, region, d.Id(), &service).Do()
	if err != nil {
		return fmt.Errorf("Error updating backend service: %s", err)
	}

	d.SetId(service.Name)

	err = computeOperationWaitRegion(config, op, project, region, "Updating Backend Service")
	if err != nil {
		return err
	}

	return resourceComputeRegionBackendServiceRead(d, meta)
}

func resourceComputeRegionBackendServiceDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	project, err := getProject(d, config)
	if err != nil {
		return err
	}

	region, err := getRegion(d, config)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Deleting backend service %s", d.Id())
	op, err := config.clientCompute.RegionBackendServices.Delete(
		project, region, d.Id()).Do()
	if err != nil {
		return fmt.Errorf("Error deleting backend service: %s", err)
	}

	err = computeOperationWaitRegion(config, op, project, region, "Deleting Backend Service")
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}

func resourceGoogleComputeRegionBackendServiceBackendHash(v interface{}) int {
	if v == nil {
		return 0
	}

	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["group"].(string)))

	if v, ok := m["description"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}

	return hashcode.String(buf.String())
}
