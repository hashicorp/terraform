package oneandone

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

func resourceOneandOnePublicIp() *schema.Resource {
	return &schema.Resource{

		Create: resourceOneandOnePublicIpCreate,
		Read:   resourceOneandOnePublicIpRead,
		Update: resourceOneandOnePublicIpUpdate,
		Delete: resourceOneandOnePublicIpDelete,
		Schema: map[string]*schema.Schema{
			"ip_type": { //IPV4 or IPV6
				Type:     schema.TypeString,
				Required: true,
			},
			"reverse_dns": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"datacenter": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceOneandOnePublicIpCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	var reverse_dns string
	var datacenter_id string

	if raw, ok := d.GetOk("reverse_dns"); ok {
		reverse_dns = raw.(string)
	}

	if raw, ok := d.GetOk("datacenter"); ok {
		dcs, err := config.API.ListDatacenters()

		if err != nil {
			return fmt.Errorf("An error occured while fetching list of datacenters %s", err)

		}

		decenter := raw.(string)
		for _, dc := range dcs {
			if strings.ToLower(dc.CountryCode) == strings.ToLower(decenter) {
				datacenter_id = dc.Id
				break
			}
		}

	}

	ip_id, ip, err := config.API.CreatePublicIp(d.Get("ip_type").(string), reverse_dns, datacenter_id)
	if err != nil {
		return err
	}

	err = config.API.WaitForState(ip, "ACTIVE", 10, config.Retries)
	if err != nil {
		return err
	}
	d.SetId(ip_id)

	return resourceOneandOnePublicIpRead(d, meta)
}

func resourceOneandOnePublicIpRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ip, err := config.API.GetPublicIp(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("ip_address", ip.IpAddress)
	d.Set("revers_dns", ip.ReverseDns)
	d.Set("datacenter", ip.Datacenter.CountryCode)
	d.Set("ip_type", ip.Type)

	return nil
}

func resourceOneandOnePublicIpUpdate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	if d.HasChange("reverse_dns") {
		_, n := d.GetChange("reverse_dns")
		ip, err := config.API.UpdatePublicIp(d.Id(), n.(string))
		if err != nil {
			return err
		}

		err = config.API.WaitForState(ip, "ACTIVE", 10, config.Retries)
		if err != nil {
			return err
		}
	}

	return resourceOneandOnePublicIpRead(d, meta)
}

func resourceOneandOnePublicIpDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	ip, err := config.API.DeletePublicIp(d.Id())
	if err != nil {
		return err
	}

	err = config.API.WaitUntilDeleted(ip)
	if err != nil {

		return err
	}

	return nil
}
