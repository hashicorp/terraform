package profitbricks

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/profitbricks/profitbricks-sdk-go"
)

func resourceProfitBricksFirewall() *schema.Resource {
	return &schema.Resource{
		Create: resourceProfitBricksFirewallCreate,
		Read:   resourceProfitBricksFirewallRead,
		Update: resourceProfitBricksFirewallUpdate,
		Delete: resourceProfitBricksFirewallDelete,
		Schema: map[string]*schema.Schema{

			"name": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"protocol": {
				Type:     schema.TypeString,
				Required: true,
			},
			"source_mac": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"source_ip": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"target_ip": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"port_range_start": {
				Type:     schema.TypeInt,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					if v.(int) < 1 && v.(int) > 65534 {
						errors = append(errors, fmt.Errorf("Port start range must be between 1 and 65534"))
					}
					return
				},
			},

			"port_range_end": {
				Type:     schema.TypeInt,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					if v.(int) < 1 && v.(int) > 65534 {
						errors = append(errors, fmt.Errorf("Port end range must be between 1 and 65534"))
					}
					return
				},
			},
			"icmp_type": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"icmp_code": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"datacenter_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"server_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"nic_id": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceProfitBricksFirewallCreate(d *schema.ResourceData, meta interface{}) error {
	fw := profitbricks.FirewallRule{
		Properties: profitbricks.FirewallruleProperties{
			Protocol: d.Get("protocol").(string),
		},
	}

	if _, ok := d.GetOk("name"); ok {
		fw.Properties.Name = d.Get("name").(string)
	}
	if _, ok := d.GetOk("source_mac"); ok {
		fw.Properties.SourceMac = d.Get("source_mac").(string)
	}
	if _, ok := d.GetOk("source_ip"); ok {
		fw.Properties.SourceIp = d.Get("source_ip").(string)
	}
	if _, ok := d.GetOk("target_ip"); ok {
		fw.Properties.TargetIp = d.Get("target_ip").(string)
	}
	if _, ok := d.GetOk("port_range_start"); ok {
		fw.Properties.PortRangeStart = d.Get("port_range_start").(int)
	}
	if _, ok := d.GetOk("port_range_end"); ok {
		fw.Properties.PortRangeEnd = d.Get("port_range_end").(int)
	}
	if _, ok := d.GetOk("icmp_type"); ok {
		fw.Properties.IcmpType = d.Get("icmp_type").(string)
	}
	if _, ok := d.GetOk("icmp_code"); ok {
		fw.Properties.IcmpCode = d.Get("icmp_code").(string)
	}

	fw = profitbricks.CreateFirewallRule(d.Get("datacenter_id").(string), d.Get("server_id").(string), d.Get("nic_id").(string), fw)

	if fw.StatusCode > 299 {
		return fmt.Errorf("An error occured while creating a firewall rule: %s", fw.Response)
	}

	err := waitTillProvisioned(meta, fw.Headers.Get("Location"))
	if err != nil {
		return err
	}
	d.SetId(fw.Id)

	return resourceProfitBricksFirewallRead(d, meta)
}

func resourceProfitBricksFirewallRead(d *schema.ResourceData, meta interface{}) error {
	fw := profitbricks.GetFirewallRule(d.Get("datacenter_id").(string), d.Get("server_id").(string), d.Get("nic_id").(string), d.Id())

	if fw.StatusCode > 299 {
		if fw.StatusCode == 404 {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("An error occured while fetching a firewall rule  dcId: %s server_id: %s  nic_id: %s ID: %s %s", d.Get("datacenter_id").(string), d.Get("server_id").(string), d.Get("nic_id").(string), d.Id(), fw.Response)
	}

	d.Set("protocol", fw.Properties.Protocol)
	d.Set("name", fw.Properties.Name)
	d.Set("source_mac", fw.Properties.SourceMac)
	d.Set("source_ip", fw.Properties.SourceIp)
	d.Set("target_ip", fw.Properties.TargetIp)
	d.Set("port_range_start", fw.Properties.PortRangeStart)
	d.Set("port_range_end", fw.Properties.PortRangeEnd)
	d.Set("icmp_type", fw.Properties.IcmpType)
	d.Set("icmp_code", fw.Properties.IcmpCode)
	d.Set("nic_id", d.Get("nic_id").(string))

	return nil
}

func resourceProfitBricksFirewallUpdate(d *schema.ResourceData, meta interface{}) error {
	properties := profitbricks.FirewallruleProperties{}

	if d.HasChange("name") {
		_, new := d.GetChange("name")

		properties.Name = new.(string)
	}
	if d.HasChange("source_mac") {
		_, new := d.GetChange("source_mac")

		properties.SourceMac = new.(string)
	}
	if d.HasChange("source_ip") {
		_, new := d.GetChange("source_ip")

		properties.SourceIp = new.(string)
	}
	if d.HasChange("target_ip") {
		_, new := d.GetChange("target_ip")

		properties.TargetIp = new.(string)
	}
	if d.HasChange("port_range_start") {
		_, new := d.GetChange("port_range_start")

		properties.PortRangeStart = new.(int)
	}
	if d.HasChange("port_range_end") {
		_, new := d.GetChange("port_range_end")

		properties.PortRangeEnd = new.(int)
	}
	if d.HasChange("icmp_type") {
		_, new := d.GetChange("icmp_type")

		properties.IcmpType = new.(int)
	}
	if d.HasChange("icmp_code") {
		_, new := d.GetChange("icmp_code")

		properties.IcmpCode = new.(int)
	}

	resp := profitbricks.PatchFirewallRule(d.Get("datacenter_id").(string), d.Get("server_id").(string), d.Get("nic_id").(string), d.Id(), properties)

	if resp.StatusCode > 299 {
		return fmt.Errorf("An error occured while deleting a firewall rule ID %s %s", d.Id(), resp.Response)
	}

	err := waitTillProvisioned(meta, resp.Headers.Get("Location"))
	if err != nil {
		return err
	}
	return resourceProfitBricksFirewallRead(d, meta)
}

func resourceProfitBricksFirewallDelete(d *schema.ResourceData, meta interface{}) error {
	resp := profitbricks.DeleteFirewallRule(d.Get("datacenter_id").(string), d.Get("server_id").(string), d.Get("nic_id").(string), d.Id())

	if resp.StatusCode > 299 {
		return fmt.Errorf("An error occured while deleting a firewall rule ID %s %s", d.Id(), string(resp.Body))
	}

	err := waitTillProvisioned(meta, resp.Headers.Get("Location"))
	if err != nil {
		return err
	}
	d.SetId("")

	return nil
}
