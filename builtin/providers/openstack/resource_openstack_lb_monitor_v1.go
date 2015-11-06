package openstack

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/monitors"
)

func resourceLBMonitorV1() *schema.Resource {
	return &schema.Resource{
		Create: resourceLBMonitorV1Create,
		Read:   resourceLBMonitorV1Read,
		Update: resourceLBMonitorV1Update,
		Delete: resourceLBMonitorV1Delete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFuncAllowMissing("OS_REGION_NAME"),
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"delay": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
			"timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
			"max_retries": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
			"url_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"http_method": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"expected_codes": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"admin_state_up": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
				Computed: true,
			},
		},
	}
}

func resourceLBMonitorV1Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	createOpts := monitors.CreateOpts{
		TenantID:      d.Get("tenant_id").(string),
		Type:          d.Get("type").(string),
		Delay:         d.Get("delay").(int),
		Timeout:       d.Get("timeout").(int),
		MaxRetries:    d.Get("max_retries").(int),
		URLPath:       d.Get("url_path").(string),
		ExpectedCodes: d.Get("expected_codes").(string),
		HTTPMethod:    d.Get("http_method").(string),
	}

	asuRaw := d.Get("admin_state_up").(string)
	if asuRaw != "" {
		asu, err := strconv.ParseBool(asuRaw)
		if err != nil {
			return fmt.Errorf("admin_state_up, if provided, must be either 'true' or 'false'")
		}
		createOpts.AdminStateUp = &asu
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	m, err := monitors.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack LB Monitor: %s", err)
	}
	log.Printf("[INFO] LB Monitor ID: %s", m.ID)

	d.SetId(m.ID)

	return resourceLBMonitorV1Read(d, meta)
}

func resourceLBMonitorV1Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	m, err := monitors.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "LB monitor")
	}

	log.Printf("[DEBUG] Retreived OpenStack LB Monitor %s: %+v", d.Id(), m)

	d.Set("type", m.Type)
	d.Set("delay", m.Delay)
	d.Set("timeout", m.Timeout)
	d.Set("max_retries", m.MaxRetries)
	d.Set("tenant_id", m.TenantID)
	d.Set("url_path", m.URLPath)
	d.Set("http_method", m.HTTPMethod)
	d.Set("expected_codes", m.ExpectedCodes)
	d.Set("admin_state_up", strconv.FormatBool(m.AdminStateUp))

	return nil
}

func resourceLBMonitorV1Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	updateOpts := monitors.UpdateOpts{
		Delay:         d.Get("delay").(int),
		Timeout:       d.Get("timeout").(int),
		MaxRetries:    d.Get("max_retries").(int),
		URLPath:       d.Get("url_path").(string),
		HTTPMethod:    d.Get("http_method").(string),
		ExpectedCodes: d.Get("expected_codes").(string),
	}

	if d.HasChange("admin_state_up") {
		asuRaw := d.Get("admin_state_up").(string)
		if asuRaw != "" {
			asu, err := strconv.ParseBool(asuRaw)
			if err != nil {
				return fmt.Errorf("admin_state_up, if provided, must be either 'true' or 'false'")
			}
			updateOpts.AdminStateUp = &asu
		}
	}

	log.Printf("[DEBUG] Updating OpenStack LB Monitor %s with options: %+v", d.Id(), updateOpts)

	_, err = monitors.Update(networkingClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack LB Monitor: %s", err)
	}

	return resourceLBMonitorV1Read(d, meta)
}

func resourceLBMonitorV1Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(d.Get("region").(string))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	err = monitors.Delete(networkingClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack LB Monitor: %s", err)
	}

	d.SetId("")
	return nil
}
