package openstack

import (
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/lbaas/vips"
)

func resourceLBVip() *schema.Resource {
	return &schema.Resource{
		Create: resourceLBVipCreate,
		Read:   resourceLBVipRead,
		Update: resourceLBVipUpdate,
		Delete: resourceLBVipDelete,

		Schema: map[string]*schema.Schema{
			"region": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				DefaultFunc: envDefaultFunc("OS_REGION_NAME"),
			},
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"pool_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"tenant_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"persistence": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: false,
			},
			"conn_limit": &schema.Schema{
				Type:     schema.TypeInt,
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

func resourceLBVipCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := openstack.NewNetworkV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	createOpts := vips.CreateOpts{
		Name:         d.Get("name").(string),
		SubnetID:     d.Get("subnet_id").(string),
		Protocol:     d.Get("protocol").(string),
		ProtocolPort: d.Get("port").(int),
		PoolID:       d.Get("pool_id").(string),
		TenantID:     d.Get("tenant_id").(string),
		Address:      d.Get("address").(string),
		Description:  d.Get("description").(string),
		Persistence:  resourceVipPersistence(d),
		ConnLimit:    gophercloud.MaybeInt(d.Get("conn_limit").(int)),
	}

	asuRaw := d.Get("admin_state_up").(string)
	if asuRaw != "" {
		asu, err := strconv.ParseBool(asuRaw)
		if err != nil {
			return fmt.Errorf("admin_state_up, if provided, must be either 'true' or 'false'")
		}
		createOpts.AdminStateUp = &asu
	}

	log.Printf("[INFO] Requesting lb vip creation")
	p, err := vips.Create(networkingClient, createOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error creating OpenStack LB VIP: %s", err)
	}
	log.Printf("[INFO] LB VIP ID: %s", p.ID)

	d.SetId(p.ID)

	return resourceLBVipRead(d, meta)
}

func resourceLBVipRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := openstack.NewNetworkV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	p, err := vips.Get(networkingClient, d.Id()).Extract()
	if err != nil {
		return fmt.Errorf("Error retrieving OpenStack LB VIP: %s", err)
	}

	log.Printf("[DEBUG] Retreived OpenStack LB VIP %s: %+v", d.Id(), p)

	d.Set("name", p.Name)
	d.Set("subnet_id", p.SubnetID)
	d.Set("protocol", p.Protocol)
	d.Set("port", p.ProtocolPort)
	d.Set("pool_id", p.PoolID)

	if _, exists := d.GetOk("tenant_id"); exists {
		if d.HasChange("tenant_id") {
			d.Set("tenant_id", p.TenantID)
		}
	} else {
		d.Set("tenant_id", "")
	}

	if _, exists := d.GetOk("address"); exists {
		if d.HasChange("address") {
			d.Set("address", p.Address)
		}
	} else {
		d.Set("address", "")
	}

	if _, exists := d.GetOk("description"); exists {
		if d.HasChange("description") {
			d.Set("description", p.Description)
		}
	} else {
		d.Set("description", "")
	}

	if _, exists := d.GetOk("persistence"); exists {
		if d.HasChange("persistence") {
			d.Set("persistence", p.Description)
		}
	}

	if _, exists := d.GetOk("conn_limit"); exists {
		if d.HasChange("conn_limit") {
			d.Set("conn_limit", p.ConnLimit)
		}
	} else {
		d.Set("conn_limit", "")
	}

	if _, exists := d.GetOk("admin_state_up"); exists {
		if d.HasChange("admin_state_up") {
			d.Set("admin_state_up", strconv.FormatBool(p.AdminStateUp))
		}
	} else {
		d.Set("admin_state_up", "")
	}

	return nil
}

func resourceLBVipUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := openstack.NewNetworkV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	var updateOpts vips.UpdateOpts
	if d.HasChange("name") {
		updateOpts.Name = d.Get("name").(string)
	}
	if d.HasChange("pool_id") {
		updateOpts.PoolID = d.Get("pool_id").(string)
	}
	if d.HasChange("description") {
		updateOpts.Description = d.Get("description").(string)
	}
	if d.HasChange("persistence") {
		updateOpts.Persistence = resourceVipPersistence(d)
	}
	if d.HasChange("conn_limit") {
		updateOpts.ConnLimit = gophercloud.MaybeInt(d.Get("conn_limit").(int))
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

	log.Printf("[DEBUG] Updating OpenStack LB VIP %s with options: %+v", d.Id(), updateOpts)

	_, err = vips.Update(networkingClient, d.Id(), updateOpts).Extract()
	if err != nil {
		return fmt.Errorf("Error updating OpenStack LB VIP: %s", err)
	}

	return resourceLBVipRead(d, meta)
}

func resourceLBVipDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)
	networkingClient, err := openstack.NewNetworkV2(config.osClient, gophercloud.EndpointOpts{
		Region: d.Get("region").(string),
	})
	if err != nil {
		return fmt.Errorf("Error creating OpenStack networking client: %s", err)
	}

	err = vips.Delete(networkingClient, d.Id()).ExtractErr()
	if err != nil {
		return fmt.Errorf("Error deleting OpenStack LB VIP: %s", err)
	}

	d.SetId("")
	return nil
}

func resourceVipPersistence(d *schema.ResourceData) *vips.SessionPersistence {
	rawP := d.Get("persistence").(interface{})
	rawMap := rawP.(map[string]interface{})
	p := vips.SessionPersistence{
		Type:       rawMap["type"].(string),
		CookieName: rawMap["cookie_name"].(string),
	}
	return &p
}
