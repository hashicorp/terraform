package vcd

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/opencredo/vmware-govcd"
	"regexp"
	"time"
)

func resourceVcdSNAT() *schema.Resource {
	return &schema.Resource{
		Create: resourceVcdSNATCreate,
		Update: resourceVcdSNATUpdate,
		Delete: resourceVcdSNATDelete,
		Read:   resourceVcdSNATRead,

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

			"internal_ip": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceVcdSNATCreate(d *schema.ResourceData, meta interface{}) error {
	vcd_client := meta.(*govcd.VCDClient)
	// Multiple VCD components need to run operations on the Edge Gateway, as
	// the edge gatway will throw back an error if it is already performing an
	// operation we must wait until we can aquire a lock on the client
	vcd_client.Mutex.Lock()
	defer vcd_client.Mutex.Unlock()
	var task govcd.Task

	// Creating a loop to offer further protection from the edge gateway erroring
	// due to being busy eg another person is using another client so wouldn't be
	// constrained by out lock. If the edge gateway reurns with a busy error, wait
	// 3 seconds and then try again. Continue until a non-busy error or success
	for {
		err := vcd_client.OrgVdc.Refresh()
		if err != nil {
			return fmt.Errorf("Error refreshing vdc: %#v", err)
		}

		edgeGateway, err := vcd_client.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))

		if err != nil {
			return fmt.Errorf("Unable to find edge gateway: %#v", err)
		}

		task, err = edgeGateway.AddNATMapping("SNAT", d.Get("internal_ip").(string),
			d.Get("external_ip").(string),
			"any")

		if err != nil {
			if v, _ := regexp.MatchString("is busy completing an operation.$", err.Error()); v {
				time.Sleep(3 * time.Second)
				continue
			} else {
				return fmt.Errorf("Error setting SNAT rules: %#v", err)
			}
		}
		break
	}

	err := task.WaitTaskCompletion()
	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}

	d.SetId(d.Get("internal_ip").(string))
	return nil
}

func resourceVcdSNATUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceVcdSNATRead(d *schema.ResourceData, meta interface{}) error {
	vcd_client := meta.(*govcd.VCDClient)
	e, err := vcd_client.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))

	if err != nil {
		return fmt.Errorf("Unable to find edge gateway: %#v", err)
	}

	var found bool

	for _, r := range e.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.NatService.NatRule {
		if r.RuleType == "SNAT" &&
			r.GatewayNatRule.OriginalIP == d.Id() {
			found = true
			d.Set("external_ip", r.GatewayNatRule.TranslatedIP)
		}
	}

	if !found {
		d.SetId("")
	}

	return nil
}

func resourceVcdSNATDelete(d *schema.ResourceData, meta interface{}) error {
	vcd_client := meta.(*govcd.VCDClient)
	// Multiple VCD components need to run operations on the Edge Gateway, as
	// the edge gatway will throw back an error if it is already performing an
	// operation we must wait until we can aquire a lock on the client
	vcd_client.Mutex.Lock()
	defer vcd_client.Mutex.Unlock()
	var task govcd.Task

	// Creating a loop to offer further protection from the edge gateway erroring
	// due to being busy eg another person is using another client so wouldn't be
	// constrained by out lock. If the edge gateway reurns with a busy error, wait
	// 3 seconds and then try again. Continue until a non-busy error or success
	for {
		err := vcd_client.OrgVdc.Refresh()
		if err != nil {
			return fmt.Errorf("Error refreshing vdc: %#v", err)
		}

		edgeGateway, err := vcd_client.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))

		if err != nil {
			return fmt.Errorf("Unable to find edge gateway: %#v", err)
		}

		task, err = edgeGateway.RemoveNATMapping("SNAT", d.Get("internal_ip").(string),
			d.Get("external_ip").(string),
			"")

		if err != nil {
			if v, _ := regexp.MatchString("is busy completing an operation.$", err.Error()); v {
				time.Sleep(3 * time.Second)
				continue
			} else {
				return fmt.Errorf("Error setting SNAT rules: %#v", err)
			}
		}
		break
	}

	err := task.WaitTaskCompletion()
	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}
	return nil
}
