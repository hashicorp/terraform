package cloudstack

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackIPAddress() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackIPAddressCreate,
		Read:   resourceCloudStackIPAddressRead,
		Delete: resourceCloudStackIPAddressDelete,

		Schema: map[string]*schema.Schema{
			"network_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"zone_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudStackIPAddressCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	if err := verifyIPAddressParams(d); err != nil {
		return err
	}

	// Create a new parameter struct
	p := cs.Address.NewAssociateIpAddressParams()

	if networkid, ok := d.GetOk("network_id"); ok {
		// Set the networkid
		p.SetNetworkid(networkid.(string))
	}

	if vpcid, ok := d.GetOk("vpc_id"); ok {
		// Set the vpcid
		p.SetVpcid(vpcid.(string))
	}

	if zoneid, ok := d.GetOk("zone_id"); ok {
		// Set the vpcid
		p.SetZoneid(zoneid.(string))
	}

	// If there is a project supplied, we retrieve and set the project id
	if err := setProjectid(p, cs, d); err != nil {
		return err
	}

	// Associate a new IP address
	r, err := cs.Address.AssociateIpAddress(p)
	if err != nil {
		return fmt.Errorf("Error associating a new IP address: %s", err)
	}

	d.SetId(r.Id)

	return resourceCloudStackIPAddressRead(d, meta)
}

func resourceCloudStackIPAddressRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the IP address details
	ip, count, err := cs.Address.GetPublicIpAddressByID(
		d.Id(),
		cloudstack.WithProject(d.Get("project").(string)),
	)
	if err != nil {
		if count == 0 {
			log.Printf(
				"[DEBUG] IP address with ID %s is no longer associated", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	// Updated the IP address
	d.Set("ip_address", ip.Ipaddress)

	if _, ok := d.GetOk("network_id"); ok {
		d.Set("network_id", ip.Associatednetworkid)
	}

	if _, ok := d.GetOk("vpc_id"); ok {
		d.Set("vpc_id", ip.Vpcid)
	}

	if _, ok := d.GetOk("zone_id"); ok {
		d.Set("zone_id", ip.Zoneid)
	}

	setValueOrID(d, "project", ip.Project, ip.Projectid)

	return nil
}

func resourceCloudStackIPAddressDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.Address.NewDisassociateIpAddressParams(d.Id())

	// Disassociate the IP address
	if _, err := cs.Address.DisassociateIpAddress(p); err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error disassociating IP address %s: %s", d.Get("name").(string), err)
	}

	return nil
}

func verifyIPAddressParams(d *schema.ResourceData) error {
	_, network := d.GetOk("network_id")
	_, vpc := d.GetOk("vpc_id")

	if (network && vpc) || (!network && !vpc) {
		return fmt.Errorf(
			"You must supply a value for either (so not both) the 'network_id' or 'vpc_id' parameter")
	}

	return nil
}
