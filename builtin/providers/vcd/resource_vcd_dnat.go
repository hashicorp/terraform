package vcd

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hmrc/vmware-govcd"
	"strings"
)

func resourceVcdDNAT() *schema.Resource {
	return &schema.Resource{
		Create: resourceVcdDNATCreate,
		Update: resourceVcdDNATUpdate,
		Delete: resourceVcdDNATDelete,
		Read:   resourceVcdDNATRead,

		Schema: map[string]*schema.Schema{
			"edge_gateway": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"external_ip": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},

			"internal_ip": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceVcdDNATCreate(d *schema.ResourceData, meta interface{}) error {
	vcd_client := meta.(*govcd.VCDClient)
	// Multiple VCD components need to run operations on the Edge Gateway, as
	// the edge gatway will throw back an error if it is already performing an
	// operation we must wait until we can aquire a lock on the client
	vcd_client.Mutex.Lock()
	defer vcd_client.Mutex.Unlock()
	portString := getPortString(d.Get("port").(int))

	edgeGateway, err := vcd_client.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))

	if err != nil {
		return fmt.Errorf("Unable to find edge gateway: %#v", err)
	}

	// Creating a loop to offer further protection from the edge gateway erroring
	// due to being busy eg another person is using another client so wouldn't be
	// constrained by out lock. If the edge gateway reurns with a busy error, wait
	// 3 seconds and then try again. Continue until a non-busy error or success

	err = retryCall(4, func() error {
		task, err := edgeGateway.AddNATMapping("DNAT", d.Get("external_ip").(string),
			d.Get("internal_ip").(string),
			portString)
		if err != nil {
			return fmt.Errorf("Error setting DNAT rules: %#v", err)
		}

		return task.WaitTaskCompletion()
	})

	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}

	d.SetId(d.Get("external_ip").(string) + "_" + portString)
	return nil
}

func resourceVcdDNATUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceVcdDNATRead(d *schema.ResourceData, meta interface{}) error {
	vcd_client := meta.(*govcd.VCDClient)
	e, err := vcd_client.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))

	if err != nil {
		return fmt.Errorf("Unable to find edge gateway: %#v", err)
	}

	idSplit := strings.Split(d.Id(), "_")
	var found bool

	for _, r := range e.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.NatService.NatRule {
		if r.RuleType == "DNAT" &&
			r.GatewayNatRule.OriginalIP == idSplit[0] &&
			r.GatewayNatRule.OriginalPort == idSplit[1] {
			found = true
			d.Set("internal_ip", r.GatewayNatRule.TranslatedIP)
		}
	}

	if !found {
		d.SetId("")
	}

	return nil
}

func resourceVcdDNATDelete(d *schema.ResourceData, meta interface{}) error {
	vcd_client := meta.(*govcd.VCDClient)
	// Multiple VCD components need to run operations on the Edge Gateway, as
	// the edge gatway will throw back an error if it is already performing an
	// operation we must wait until we can aquire a lock on the client
	vcd_client.Mutex.Lock()
	defer vcd_client.Mutex.Unlock()
	portString := getPortString(d.Get("port").(int))

	edgeGateway, err := vcd_client.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))

	if err != nil {
		return fmt.Errorf("Unable to find edge gateway: %#v", err)
	}
	err = retryCall(4, func() error {
		task, err := edgeGateway.RemoveNATMapping("DNAT", d.Get("external_ip").(string),
			d.Get("internal_ip").(string),
			portString)
		if err != nil {
			return fmt.Errorf("Error setting DNAT rules: %#v", err)
		}

		return task.WaitTaskCompletion()
	})
	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}
	return nil
}
