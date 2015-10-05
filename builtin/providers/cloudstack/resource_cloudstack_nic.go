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
			"network": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ipaddress": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"virtual_machine": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceCloudStackNICCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Retrieve the network ID
	networkid, e := retrieveID(cs, "network", d.Get("network").(string))
	if e != nil {
		return e.Error()
	}

	// Retrieve the virtual_machine ID
	virtualmachineid, e := retrieveID(cs, "virtual_machine", d.Get("virtual_machine").(string))
	if e != nil {
		return e.Error()
	}

	// Create a new parameter struct
	p := cs.VirtualMachine.NewAddNicToVirtualMachineParams(networkid, virtualmachineid)

	// If there is a ipaddres supplied, add it to the parameter struct
	if ipaddress, ok := d.GetOk("ipaddress"); ok {
		p.SetIpaddress(ipaddress.(string))
	}

	// Create and attach the new NIC
	r, err := cs.VirtualMachine.AddNicToVirtualMachine(p)
	if err != nil {
		return fmt.Errorf("Error creating the new NIC: %s", err)
	}

	found := false
	for _, n := range r.Nic {
		if n.Networkid == networkid {
			d.SetId(n.Id)
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("Could not find NIC ID for network: %s", d.Get("network").(string))
	}

	return resourceCloudStackNICRead(d, meta)
}

func resourceCloudStackNICRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the virtual machine details
	vm, count, err := cs.VirtualMachine.GetVirtualMachineByName(d.Get("virtual_machine").(string))
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] Instance %s does no longer exist", d.Get("virtual_machine").(string))
			d.SetId("")
			return nil
		} else {
			return err
		}
	}

	// Read NIC info
	found := false
	for _, n := range vm.Nic {
		if n.Id == d.Id() {
			d.Set("ipaddress", n.Ipaddress)
			setValueOrID(d, "network", n.Networkname, n.Networkid)
			setValueOrID(d, "virtual_machine", vm.Name, vm.Id)
			found = true
			break
		}
	}

	if !found {
		log.Printf("[DEBUG] NIC for network %s does no longer exist", d.Get("network").(string))
		d.SetId("")
	}

	return nil
}

func resourceCloudStackNICDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Retrieve the virtual_machine ID
	virtualmachineid, e := retrieveID(cs, "virtual_machine", d.Get("virtual_machine").(string))
	if e != nil {
		return e.Error()
	}

	// Create a new parameter struct
	p := cs.VirtualMachine.NewRemoveNicFromVirtualMachineParams(d.Id(), virtualmachineid)

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
