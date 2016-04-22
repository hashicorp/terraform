package cloudstack

import (
	"errors"
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
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"network": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "Please use the `network_id` field instead",
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"ipaddress": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "Please use the `ip_address` field instead",
			},

			"virtual_machine_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"virtual_machine": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "Please use the `virtual_machine_id` field instead",
			},
		},
	}
}

func resourceCloudStackNICCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	network, ok := d.GetOk("network_id")
	if !ok {
		network, ok = d.GetOk("network")
	}
	if !ok {
		return errors.New("Either `network_id` or [deprecated] `network` must be provided.")
	}

	// Retrieve the network ID
	networkid, e := retrieveID(cs, "network", network.(string))
	if e != nil {
		return e.Error()
	}

	virtualmachine, ok := d.GetOk("virtual_machine_id")
	if !ok {
		virtualmachine, ok = d.GetOk("virtual_machine")
	}
	if !ok {
		return errors.New(
			"Either `virtual_machine_id` or [deprecated] `virtual_machine` must be provided.")
	}

	// Retrieve the virtual_machine ID
	virtualmachineid, e := retrieveID(cs, "virtual_machine", virtualmachine.(string))
	if e != nil {
		return e.Error()
	}

	// Create a new parameter struct
	p := cs.VirtualMachine.NewAddNicToVirtualMachineParams(networkid, virtualmachineid)

	// If there is a ipaddres supplied, add it to the parameter struct
	ipaddress, ok := d.GetOk("ip_address")
	if !ok {
		ipaddress, ok = d.GetOk("ipaddress")
	}
	if ok {
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
		return fmt.Errorf("Could not find NIC ID for network ID: %s", networkid)
	}

	return resourceCloudStackNICRead(d, meta)
}

func resourceCloudStackNICRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	virtualmachine, ok := d.GetOk("virtual_machine_id")
	if !ok {
		virtualmachine, ok = d.GetOk("virtual_machine")
	}
	if !ok {
		return errors.New(
			"Either `virtual_machine_id` or [deprecated] `virtual_machine` must be provided.")
	}

	// Retrieve the virtual_machine ID
	virtualmachineid, e := retrieveID(cs, "virtual_machine", virtualmachine.(string))
	if e != nil {
		return e.Error()
	}

	// Get the virtual machine details
	vm, count, err := cs.VirtualMachine.GetVirtualMachineByID(virtualmachineid)
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] Instance %s does no longer exist", d.Get("virtual_machine").(string))
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

	virtualmachine, ok := d.GetOk("virtual_machine_id")
	if !ok {
		virtualmachine, ok = d.GetOk("virtual_machine")
	}
	if !ok {
		return errors.New(
			"Either `virtual_machine_id` or [deprecated] `virtual_machine` must be provided.")
	}

	// Retrieve the virtual_machine ID
	virtualmachineid, e := retrieveID(cs, "virtual_machine", virtualmachine.(string))
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
