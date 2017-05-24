package profitbricks

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
)

func resourceProfitBricksLoadbalancer() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfitBricksLoadbalancerCreate,
		Read:   resourceProfitBricksLoadbalancerRead,
		Update: resourceProfitBricksLoadbalancerUpdate,
		Delete: resourceProfitBricksLoadbalancerDelete,
		Schema: map[string]*schema.Schema{

			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"dhcp": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"datacenter_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"nic_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceProfitBricksLoadbalancerCreate(d *schema.ResourceData, meta interface{}) error {
	lb := profitbricks.Loadbalancer{
		Properties: profitbricks.LoadbalancerProperties{
			Name: d.Get("name").(string),
		},
	}

	lb = profitbricks.CreateLoadbalancer(d.Get("datacenter_id").(string), lb)

	if lb.StatusCode > 299 {
		return fmt.Errorf("Error occured while creating a loadbalancer %s", lb.Response)
	}
	err := waitTillProvisioned(meta, lb.Headers.Get("Location"))

	if err != nil {
		return err
	}

	d.SetId(lb.Id)

	nic := profitbricks.AssociateNic(d.Get("datacenter_id").(string), d.Id(), d.Get("nic_id").(string))

	if nic.StatusCode > 299 {
		return fmt.Errorf("Error occured while deleting a balanced nic: %s", nic.Response)
	}
	err = waitTillProvisioned(meta, nic.Headers.Get("Location"))
	if err != nil {
		return err
	}

	return resourceProfitBricksLoadbalancerRead(d, meta)
}

func resourceProfitBricksLoadbalancerRead(d *schema.ResourceData, meta interface{}) error {
	lb := profitbricks.GetLoadbalancer(d.Get("datacenter_id").(string), d.Id())

	if lb.StatusCode > 299 {
		if lb.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("An error occured while fetching a lan ID %s %s", d.Id(), lb.Response)
	}

	d.Set("name", lb.Properties.Name)
	d.Set("ip", lb.Properties.Ip)
	d.Set("dhcp", lb.Properties.Dhcp)

	return nil
}

func resourceProfitBricksLoadbalancerUpdate(d *schema.ResourceData, meta interface{}) error {
	properties := profitbricks.LoadbalancerProperties{}
	if d.HasChange("name") {
		_, new := d.GetChange("name")
		properties.Name = new.(string)
	}
	if d.HasChange("ip") {
		_, new := d.GetChange("ip")
		properties.Ip = new.(string)
	}
	if d.HasChange("dhcp") {
		_, new := d.GetChange("dhcp")
		properties.Dhcp = new.(bool)
	}

	if d.HasChange("nic_id") {
		old, new := d.GetChange("dhcp")

		resp := profitbricks.DeleteBalancedNic(d.Get("datacenter_id").(string), d.Id(), old.(string))
		if resp.StatusCode > 299 {
			return fmt.Errorf("Error occured while deleting a balanced nic: %s", string(resp.Body))
		}
		err := waitTillProvisioned(meta, resp.Headers.Get("Location"))
		if err != nil {
			return err
		}

		nic := profitbricks.AssociateNic(d.Get("datacenter_id").(string), d.Id(), new.(string))
		if nic.StatusCode > 299 {
			return fmt.Errorf("Error occured while deleting a balanced nic: %s", nic.Response)
		}
		err = waitTillProvisioned(meta, nic.Headers.Get("Location"))
		if err != nil {
			return err
		}
	}

	return resourceProfitBricksLoadbalancerRead(d, meta)
}

func resourceProfitBricksLoadbalancerDelete(d *schema.ResourceData, meta interface{}) error {
	resp := profitbricks.DeleteLoadbalancer(d.Get("datacenter_id").(string), d.Id())

	if resp.StatusCode > 299 {
		return fmt.Errorf("Error occured while deleting a loadbalancer: %s", string(resp.Body))
	}

	err := waitTillProvisioned(meta, resp.Headers.Get("Location"))
	if err != nil {
		return err
	}

	d.SetId("")
	return nil
}
