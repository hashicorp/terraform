package openstack

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/lbaas_v2/monitors"
)

func resourceMonitorV2() *schema.Resource {
	return &schema.Resource{
		Create: resourceMonitorV2Create,
		Read:   resourceMonitorV2Read,
		Update: resourceMonitorV2Update,
		Delete: resourceMonitorV2Delete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: schema.EnvDefaultFunc("OS_REGION_NAME", ""),
			},

			"pool_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
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
			},
			"timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"max_retries": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"url_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"http_method": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"expected_codes": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"admin_state_up": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},

			"id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceMonitorV2Create(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	adminStateUp := d.Get("admin_state_up").(bool)
	createOpts := monitors.CreateOpts{
		PoolID:        d.Get("pool_id").(string),
		TenantID:      d.Get("tenant_id").(string),
		Type:          d.Get("type").(string),
		Delay:         d.Get("delay").(int),
		Timeout:       d.Get("timeout").(int),
		MaxRetries:    d.Get("max_retries").(int),
		URLPath:       d.Get("url_path").(string),
		HTTPMethod:    d.Get("http_method").(string),
		ExpectedCodes: d.Get("expected_codes").(string),
		Name:          d.Get("name").(string),
		AdminStateUp:  &adminStateUp,
	}

	log.Printf("[DEBUG] Create Options: %#v", createOpts)
	monitor, err := monitors.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack LBaaSV2 monitor: %s", err)
	}
	log.Printf("[INFO] monitor ID: %s", monitor.ID)

	log.Printf("[DEBUG] Waiting for Openstack LBaaSV2 monitor (%s) to become available.", monitor.ID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"PENDING_CREATE"},
		Target:     []string{"ACTIVE"},
		Refresh:    waitForMonitorActive(networkingClient, monitor.ID),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	d.SetId(monitor.ID)

	return resourceMonitorV2Read(d, meta)
}

func resourceMonitorV2Read(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	monitor, err := monitors.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return CheckDeleted(d, err, "LBV2 Monitor")
	}

	log.Printf("[DEBUG] Retrieved OpenStack LBaaSV2 Monitor %s: %+v", d.Id(), monitor)

	d.Set("id", monitor.ID)
	d.Set("tenant_id", monitor.TenantID)
	d.Set("type", monitor.Type)
	d.Set("delay", monitor.Delay)
	d.Set("timeout", monitor.Timeout)
	d.Set("max_retries", monitor.MaxRetries)
	d.Set("url_path", monitor.URLPath)
	d.Set("http_method", monitor.HTTPMethod)
	d.Set("expected_codes", monitor.ExpectedCodes)
	d.Set("admin_state_up", monitor.AdminStateUp)
	d.Set("name", monitor.Name)

	return nil
}

func resourceMonitorV2Update(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var updateOpts monitors.UpdateOpts
	if d.HasChange("url_path") {
		updateOpts.URLPath = d.Get("url_path").(string)
	}
	if d.HasChange("expected_codes") {
		updateOpts.ExpectedCodes = d.Get("expected_codes").(string)
	}
	if d.HasChange("delay") {
		updateOpts.Delay = d.Get("delay").(int)
	}
	if d.HasChange("timeout") {
		updateOpts.Timeout = d.Get("timeout").(int)
	}
	if d.HasChange("max_retries") {
		updateOpts.MaxRetries = d.Get("max_retries").(int)
	}
	if d.HasChange("admin_state_up") {
		asu := d.Get("admin_state_up").(bool)
		updateOpts.AdminStateUp = &asu
	}
	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("http_method") {
		updateOpts.HTTPMethod = d.Get("http_method").(string)
	}

	log.Printf("[DEBUG] Updating OpenStack LBaaSV2 Monitor %s with options: %+v", d.Id(), updateOpts)

	_, err = monitors.Update(networkingClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack LBaaSV2 Monitor: %s", err)
	}

	return resourceMonitorV2Read(d, meta)
}

func resourceMonitorV2Delete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := config.networkingV2Client(GetRegion(d))
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"ACTIVE", "PENDING_DELETE"},
		Target:     []string{"DELETED"},
		Refresh:    waitForMonitorDelete(networkingClient, d.Id()),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      5 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack LBaaSV2 Monitor: %s", err)
	}

	d.SetId("")
	return nil
}

func waitForMonitorActive(networkingClient *gophercloud.ServiceClient, monitorID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		monitor, err := monitors.Get(networkingClient, monitorID).Extract()
		if err != nil {
			return nil, "", err
		}

		log.Printf("[DEBUG] OpenStack LBaaSV2 Monitor: %+v", monitor)
		return monitor, "ACTIVE", nil
	}
}

func waitForMonitorDelete(networkingClient *gophercloud.ServiceClient, monitorID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		log.Printf("[DEBUG] Attempting to delete OpenStack LBaaSV2 Monitor %s", monitorID)

		monitor, err := monitors.Get(networkingClient, monitorID).Extract()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack LBaaSV2 Monitor %s", monitorID)
				return monitor, "DELETED", nil
			}
			return monitor, "ACTIVE", err
		}

		log.Printf("[DEBUG] Openstack LBaaSV2 Monitor: %+v", monitor)
		err = monitors.Delete(networkingClient, monitorID).ExtractErr()
		if err != nil {
			if _, ok := err.(gophercloud.ErrDefault404); ok {
				log.Printf("[DEBUG] Successfully deleted OpenStack LBaaSV2 Monitor %s", monitorID)
				return monitor, "DELETED", nil
			}

			if errCode, ok := err.(gophercloud.ErrUnexpectedResponseCode); ok {
				if errCode.Actual == 409 {
					log.Printf("[DEBUG] OpenStack LBaaSV2 Monitor (%s) is still in use.", monitorID)
					return monitor, "ACTIVE", nil
				}
			}

			return monitor, "ACTIVE", err
		}

		log.Printf("[DEBUG] OpenStack LBaaSV2 Monitor %s still active.", monitorID)
		return monitor, "ACTIVE", nil
	}
}
