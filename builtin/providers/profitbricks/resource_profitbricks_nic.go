package profitbricks

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
	"log"
	"strings"
)

func resourceProfitBricksNic() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfitBricksNicCreate,
		Read:   resourceProfitBricksNicRead,
		Update: resourceProfitBricksNicUpdate,
		Delete: resourceProfitBricksNicDelete,
		Schema: map[string]*schema.Schema{

			"lan": {
				Type:     schema.TypeInt,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"dhcp": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"ip": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"ips": {
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"firewall_active": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"nat": {
				Type:     schema.TypeBool,
				Optional: true,
			},
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"datacenter_id": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceProfitBricksNicCreate(d *schema.ResourceData, meta interface{}) error {
	nic := profitbricks.Nic{
		Properties: profitbricks.NicProperties{
			Lan: d.Get("lan").(int),
		},
	}
	if _, ok := d.GetOk("name"); ok {
		nic.Properties.Name = d.Get("name").(string)
	}
	if _, ok := d.GetOk("dhcp"); ok {
		nic.Properties.Dhcp = d.Get("dhcp").(bool)
	}

	if _, ok := d.GetOk("ip"); ok {
		raw := d.Get("ip").(string)
		ips := strings.Split(raw, ",")
		nic.Properties.Ips = ips
	}
	if _, ok := d.GetOk("firewall_active"); ok {
		raw := d.Get("firewall_active").(bool)
		nic.Properties.FirewallActive = raw
	}
	if _, ok := d.GetOk("nat"); ok {
		raw := d.Get("nat").(bool)
		nic.Properties.Nat = raw
	}

	nic = profitbricks.CreateNic(d.Get("datacenter_id").(string), d.Get("server_id").(string), nic)
	if nic.StatusCode > 299 {
		return fmt.Errorf("Error occured while creating a nic: %s", nic.Response)
	}

	err := waitTillProvisioned(meta, nic.Headers.Get("Location"))
	if err != nil {
		return err
	}
	resp := profitbricks.RebootServer(d.Get("datacenter_id").(string), d.Get("server_id").(string))
	if resp.StatusCode > 299 {
		return fmt.Errorf("Error occured while creating a nic: %s", string(resp.Body))

	}
	err = waitTillProvisioned(meta, resp.Headers.Get("Location"))
	if err != nil {
		return err
	}
	d.SetId(nic.Id)
	return resourceProfitBricksNicRead(d, meta)
}

func resourceProfitBricksNicRead(d *schema.ResourceData, meta interface{}) error {
	nic := profitbricks.GetNic(d.Get("datacenter_id").(string), d.Get("server_id").(string), d.Id())
	if nic.StatusCode > 299 {
		if nic.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error occured while fetching a nic ID %s %s", d.Id(), nic.Response)
	}
	log.Printf("[INFO] LAN ON NIC: %d", nic.Properties.Lan)
	d.Set("dhcp", nic.Properties.Dhcp)
	d.Set("lan", nic.Properties.Lan)
	d.Set("name", nic.Properties.Name)
	d.Set("ips", nic.Properties.Ips)

	return nil
}

func resourceProfitBricksNicUpdate(d *schema.ResourceData, meta interface{}) error {
	properties := profitbricks.NicProperties{}

	if d.HasChange("name") {
		_, n := d.GetChange("name")

		properties.Name = n.(string)
	}
	if d.HasChange("lan") {
		_, n := d.GetChange("lan")
		properties.Lan = n.(int)
	}
	if d.HasChange("dhcp") {
		_, n := d.GetChange("dhcp")
		properties.Dhcp = n.(bool)
	}
	if d.HasChange("ip") {
		_, raw := d.GetChange("ip")
		ips := strings.Split(raw.(string), ",")
		properties.Ips = ips
	}
	if d.HasChange("nat") {
		_, raw := d.GetChange("nat")
		nat := raw.(bool)
		properties.Nat = nat
	}

	nic := profitbricks.PatchNic(d.Get("datacenter_id").(string), d.Get("server_id").(string), d.Id(), properties)

	if nic.StatusCode > 299 {
		return fmt.Errorf("Error occured while updating a nic: %s", nic.Response)
	}
	err := waitTillProvisioned(meta, nic.Headers.Get("Location"))
	if err != nil {
		return err
	}
	return resourceProfitBricksNicRead(d, meta)
}

func resourceProfitBricksNicDelete(d *schema.ResourceData, meta interface{}) error {
	resp := profitbricks.DeleteNic(d.Get("datacenter_id").(string), d.Get("server_id").(string), d.Id())
	err := waitTillProvisioned(meta, resp.Headers.Get("Location"))
	if err != nil {
		return err
	}
	d.SetId("")
	return nil
}
