package cloudstack

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackNIC() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackNICCreate,
		Read:   resourceCloudStackNICRead,
		Delete: resourceCloudStackNICDelete,

		Schema: map[string]*schema.Schema{
			"network_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"virtual_machine_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceCloudStackNICCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.VirtualMachine.NewAddNicToVirtualMachineParams(
		d.Get("network_id").(string),
		d.Get("virtual_machine_id").(string),
	)

	// If there is a ipaddres supplied, add it to the parameter struct
	if ipaddress, ok := d.GetOk("ip_address"); ok {
		p.SetIpaddress(ipaddress.(string))
	}

	// Create and attach the new NIC
	r, err := Retry(10, retryableAddNicFunc(cs, p))
	if err != nil {
		return fmt.Errorf("Error creating the new NIC: %s", err)
	}

	found := false
	for _, n := range r.(*cloudstack.AddNicToVirtualMachineResponse).Nic {
		if n.Networkid == d.Get("network_id").(string) {
			d.SetId(n.Id)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("Could not find NIC ID for network ID: %s", d.Get("network_id").(string))
	}

	return resourceCloudStackNICRead(d, meta)
}

func resourceCloudStackNICRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the virtual machine details
	vm, count, err := cs.VirtualMachine.GetVirtualMachineByID(d.Get("virtual_machine_id").(string))
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] Instance %s does no longer exist", d.Get("virtual_machine_id").(string))
			d.SetId("")
			return nil
		}

		return err
	}

	// Read NIC info
	found := false
	for _, n := range vm.Nic {
		if n.Id == d.Id() {
			d.Set("ip_address", n.Ipaddress)
			d.Set("network_id", n.Networkid)
			d.Set("virtual_machine_id", vm.Id)
			found = true
			break
		}
	}

	if !found {
		log.Printf("[DEBUG] NIC for network ID %s does no longer exist", d.Get("network_id").(string))
		d.SetId("")
	}

	return nil
}

func resourceCloudStackNICDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.VirtualMachine.NewRemoveNicFromVirtualMachineParams(
		d.Id(),
		d.Get("virtual_machine_id").(string),
	)

	// Remove the NIC
	_, err := cs.VirtualMachine.RemoveNicFromVirtualMachine(p)
	if err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error deleting NIC: %s", err)
	}

	return nil
}

func retryableAddNicFunc(cs *cloudstack.CloudStackClient, p *cloudstack.AddNicToVirtualMachineParams) func() (interface{}, error) {
	return func() (interface{}, error) {
		r, err := cs.VirtualMachine.AddNicToVirtualMachine(p)
		if err != nil {
			return nil, err
		}
		return r, nil
	}
}
