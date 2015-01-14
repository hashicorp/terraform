package openstack

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/monitors"
)

func resourceLBMonitor() *schema.Resource {
	return &schema.Resource{
		Create: resourceLBMonitorCreate,
		Read:   resourceLBMonitorRead,
		Update: resourceLBMonitorUpdate,
		Delete: resourceLBMonitorDelete,

		Schema: map[string]*schema.Schema{
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
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
			},
		},
	}
}

func resourceLBMonitorCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

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

	log.Printf("[INFO] Requesting lb monitor creation")
	m, err := monitors.Create(osClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack LB Monitor: %s", err)
	}
	log.Printf("[INFO] LB Monitor ID: %s", m.ID)

	d.SetId(m.ID)

	return resourceLBMonitorRead(d, meta)
}

func resourceLBMonitorRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	m, err := monitors.Get(osClient, d.Id()).Extract()
	if err != nil {
		return fmt.Errorf("Error retrieving OpenStack LB Monitor: %s", err)
	}

	log.Printf("[DEBUG] Retreived OpenStack LB Monitor %s: %+v", d.Id(), m)

	d.Set("type", m.Type)
	d.Set("delay", m.Delay)
	d.Set("timeout", m.Timeout)
	d.Set("max_retries", m.MaxRetries)

	if _, exists := d.GetOk("tenant_id"); exists {
		if d.HasChange("tenant_id") {
			d.Set("tenant_id", m.TenantID)
		}
	} else {
		d.Set("tenant_id", "")
	}

	if _, exists := d.GetOk("url_path"); exists {
		if d.HasChange("url_path") {
			d.Set("url_path", m.URLPath)
		}
	} else {
		d.Set("url_path", "")
	}

	if _, exists := d.GetOk("http_method"); exists {
		if d.HasChange("http_method") {
			d.Set("http_method", m.HTTPMethod)
		}
	} else {
		d.Set("http_method", "")
	}

	if _, exists := d.GetOk("expected_codes"); exists {
		if d.HasChange("expected_codes") {
			d.Set("expected_codes", m.ExpectedCodes)
		}
	} else {
		d.Set("expected_codes", "")
	}

	if _, exists := d.GetOk("admin_state_up"); exists {
		if d.HasChange("admin_state_up") {
			d.Set("admin_state_up", strconv.FormatBool(m.AdminStateUp))
		}
	} else {
		d.Set("admin_state_up", "")
	}

	return nil
}

func resourceLBMonitorUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	var updateOpts monitors.UpdateOpts
	if d.HasChange("delay") {
		updateOpts.Delay = d.Get("delay").(int)
	}
	if d.HasChange("timeout") {
		updateOpts.Timeout = d.Get("timeout").(int)
	}
	if d.HasChange("max_retries") {
		updateOpts.MaxRetries = d.Get("max_retries").(int)
	}
	if d.HasChange("url_path") {
		updateOpts.URLPath = d.Get("url_path").(string)
	}
	if d.HasChange("http_method") {
		updateOpts.HTTPMethod = d.Get("http_method").(string)
	}
	if d.HasChange("expected_codes") {
		updateOpts.ExpectedCodes = d.Get("expected_codes").(string)
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

	_, err := monitors.Update(osClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack LB Monitor: %s", err)
	}

	return resourceLBMonitorRead(d, meta)
}

func resourceLBMonitorDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	osClient := config.networkingV2Client

	err := monitors.Delete(osClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack LB Monitor: %s", err)
	}

	d.SetId("")
	return nil
}
