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
				Computed: true,
				ForceNew: true,
			},

			"network": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "Please use the `network_id` field instead",
			},

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"vpc": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "Please use the `vpc_id` field instead",
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
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

	network, ok := d.GetOk("network_id")
	if !ok {
		network, ok = d.GetOk("network")
	}
	if ok {
		// Retrieve the network ID
		networkid, e := retrieveID(
			cs,
			"network",
			network.(string),
			cloudstack.WithProject(d.Get("project").(string)),
		)
		if e != nil {
			return e.Error()
		}

		// Set the networkid
		p.SetNetworkid(networkid)
	}

	vpc, ok := d.GetOk("vpc_id")
	if !ok {
		vpc, ok = d.GetOk("vpc")
	}
	if ok {
		// Retrieve the vpc ID
		vpcid, e := retrieveID(
			cs,
			"vpc",
			vpc.(string),
			cloudstack.WithProject(d.Get("project").(string)),
		)
		if e != nil {
			return e.Error()
		}

		// Set the vpcid
		p.SetVpcid(vpcid)
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

	_, networkID := d.GetOk("network_id")
	_, network := d.GetOk("network")
	if networkID || network {
		d.Set("network_id", ip.Associatednetworkid)
	}

	_, vpcID := d.GetOk("vpc_id")
	_, vpc := d.GetOk("vpc")
	if vpcID || vpc {
		d.Set("vpc_id", ip.Vpcid)
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
	_, networkID := d.GetOk("network_id")
	_, network := d.GetOk("network")
	_, vpcID := d.GetOk("vpc_id")
	_, vpc := d.GetOk("vpc")

	if (networkID || network) && (vpcID || vpc) || (!networkID && !network) && (!vpcID && !vpc) {
		return fmt.Errorf(
			"You must supply a value for either (so not both) the 'network_id' or 'vpc_id' parameter")
	}

	return nil
}
