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
			"ip_address_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network_id": &schema.Schema{
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

	ipaddressid := d.Get("ip_address_id").(string)
	virtualmachineid := d.Get("virtual_machine_id").(string)

	// Create a new parameter struct
	p := cs.NAT.NewEnableStaticNatParams(ipaddressid, virtualmachineid)

	if networkid, ok := d.GetOk("network_id"); ok {
		p.SetNetworkid(networkid.(string))
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

	d.Set("network_id", ip.Associatednetworkid)
	d.Set("virtual_machine_id", ip.Virtualmachineid)
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
