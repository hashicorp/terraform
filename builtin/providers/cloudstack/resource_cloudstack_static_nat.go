package cloudstack

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackStaticNAT() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackStaticNATCreate,
		Exists: resourceCloudStackStaticNATExists,
		Read:   resourceCloudStackStaticNATRead,
		Delete: resourceCloudStackStaticNATDelete,

		Schema: map[string]*schema.Schema{
			"ipaddress": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network": &schema.Schema{
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

			"vm_guest_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
		},
	}
}

func resourceCloudStackStaticNATCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Retrieve the ipaddress ID
	ipaddressid, e := retrieveID(cs, "ipaddress", d.Get("ipaddress").(string))
	if e != nil {
		return e.Error()
	}

	// Retrieve the virtual_machine ID
	virtualmachineid, e := retrieveID(cs, "virtual_machine", d.Get("virtual_machine").(string))
	if e != nil {
		return e.Error()
	}

	// Create a new parameter struct
	p := cs.NAT.NewEnableStaticNatParams(ipaddressid, virtualmachineid)

	if network, ok := d.GetOk("network"); ok {
		// Retrieve the network ID
		networkid, e := retrieveID(cs, "network", network.(string))
		if e != nil {
			return e.Error()
		}

		p.SetNetworkid(networkid)
	}

	if vmGuestIP, ok := d.GetOk("vm_guest_ip"); ok {
		p.SetVmguestip(vmGuestIP.(string))
	}

	_, err := cs.NAT.EnableStaticNat(p)
	if err != nil {
		return fmt.Errorf("Error enabling static NAT: %s", err)
	}

	d.SetId(ipaddressid)

	return resourceCloudStackStaticNATRead(d, meta)
}

func resourceCloudStackStaticNATExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the IP address details
	ip, count, err := cs.Address.GetPublicIpAddressByID(d.Id())
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] IP address with ID %s no longer exists", d.Id())
			return false, nil
		}

		return false, err
	}

	return ip.Isstaticnat, nil
}

func resourceCloudStackStaticNATRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the IP address details
	ip, count, err := cs.Address.GetPublicIpAddressByID(d.Id())
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] IP address with ID %s no longer exists", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	if !ip.Isstaticnat {
		log.Printf("[DEBUG] Static NAT is no longer enabled for IP address with ID %s", d.Id())
		d.SetId("")
		return nil
	}

	setValueOrID(d, "network", ip.Associatednetworkname, ip.Associatednetworkid)
	setValueOrID(d, "virtual_machine", ip.Virtualmachinename, ip.Virtualmachineid)
	d.Set("vm_guest_ip", ip.Vmipaddress)

	return nil
}

func resourceCloudStackStaticNATDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.NAT.NewDisableStaticNatParams(d.Id())

	// Disable static NAT
	_, err := cs.NAT.DisableStaticNat(p)
	if err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error disabling static NAT: %s", err)
	}

	return nil
}
