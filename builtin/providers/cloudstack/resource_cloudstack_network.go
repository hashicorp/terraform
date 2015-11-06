package cloudstack

import (
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackNetworkCreate,
		Read:   resourceCloudStackNetworkRead,
		Update: resourceCloudStackNetworkUpdate,
		Delete: resourceCloudStackNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"display_text": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"cidr": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"network_offering": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"vpc": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"aclid": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceCloudStackNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	name := d.Get("name").(string)

	// Retrieve the network_offering ID
	networkofferingid, e := retrieveID(cs, "network_offering", d.Get("network_offering").(string))
	if e != nil {
		return e.Error()
	}

	// Retrieve the zone ID
	zoneid, e := retrieveID(cs, "zone", d.Get("zone").(string))
	if e != nil {
		return e.Error()
	}

	// Compute/set the display text
	displaytext, ok := d.GetOk("display_text")
	if !ok {
		displaytext = name
	}

	// Create a new parameter struct
	p := cs.Network.NewCreateNetworkParams(displaytext.(string), name, networkofferingid, zoneid)

	// Get the network details from the CIDR
	m, err := parseCIDR(d.Get("cidr").(string))
	if err != nil {
		return err
	}

	// Set the needed IP config
	p.SetStartip(m["start"])
	p.SetGateway(m["gateway"])
	p.SetEndip(m["end"])
	p.SetNetmask(m["netmask"])

	// Check is this network needs to be created in a VPC
	vpc := d.Get("vpc").(string)
	if vpc != "" {
		// Retrieve the vpc ID
		vpcid, e := retrieveID(cs, "vpc", vpc)
		if e != nil {
			return e.Error()
		}

		// Set the vpc ID
		p.SetVpcid(vpcid)

		// Since we're in a VPC, check if we want to assiciate an ACL list
		aclid := d.Get("aclid").(string)
		if aclid != "" {
			// Set the acl ID
			p.SetAclid(aclid)
		}
	}

	// If there is a project supplied, we retrieve and set the project id
	if project, ok := d.GetOk("project"); ok {
		// Retrieve the project ID
		projectid, e := retrieveID(cs, "project", project.(string))
		if e != nil {
			return e.Error()
		}
		// Set the default project ID
		p.SetProjectid(projectid)
	}

	// Create the new network
	r, err := cs.Network.CreateNetwork(p)
	if err != nil {
		return fmt.Errorf("Error creating network %s: %s", name, err)
	}

	d.SetId(r.Id)

	return resourceCloudStackNetworkRead(d, meta)
}

func resourceCloudStackNetworkRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the virtual machine details
	n, count, err := cs.Network.GetNetworkByID(d.Id())
	if err != nil {
		if count == 0 {
			log.Printf(
				"[DEBUG] Network %s does no longer exist", d.Get("name").(string))
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", n.Name)
	d.Set("display_text", n.Displaytext)
	d.Set("cidr", n.Cidr)

	setValueOrID(d, "network_offering", n.Networkofferingname, n.Networkofferingid)
	setValueOrID(d, "project", n.Project, n.Projectid)
	setValueOrID(d, "zone", n.Zonename, n.Zoneid)

	return nil
}

func resourceCloudStackNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)
	name := d.Get("name").(string)

	// Create a new parameter struct
	p := cs.Network.NewUpdateNetworkParams(d.Id())

	// Check if the name or display text is changed
	if d.HasChange("name") || d.HasChange("display_text") {
		p.SetName(name)

		// Compute/set the display text
		displaytext := d.Get("display_text").(string)
		if displaytext == "" {
			displaytext = name
		}
		p.SetDisplaytext(displaytext)
	}

	// Check if the cidr is changed
	if d.HasChange("cidr") {
		p.SetGuestvmcidr(d.Get("cidr").(string))
	}

	// Check if the network offering is changed
	if d.HasChange("network_offering") {
		// Retrieve the network_offering ID
		networkofferingid, e := retrieveID(cs, "network_offering", d.Get("network_offering").(string))
		if e != nil {
			return e.Error()
		}
		// Set the new network offering
		p.SetNetworkofferingid(networkofferingid)
	}

	// Update the network
	_, err := cs.Network.UpdateNetwork(p)
	if err != nil {
		return fmt.Errorf(
			"Error updating network %s: %s", name, err)
	}

	return resourceCloudStackNetworkRead(d, meta)
}

func resourceCloudStackNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.Network.NewDeleteNetworkParams(d.Id())

	// Delete the network
	_, err := cs.Network.DeleteNetwork(p)
	if err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error deleting network %s: %s", d.Get("name").(string), err)
	}
	return nil
}

func parseCIDR(cidr string) (map[string]string, error) {
	m := make(map[string]string, 4)

	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse cidr %s: %s", cidr, err)
	}

	msk := ipnet.Mask
	sub := ip.Mask(msk)

	m["netmask"] = fmt.Sprintf("%d.%d.%d.%d", msk[0], msk[1], msk[2], msk[3])
	m["gateway"] = fmt.Sprintf("%d.%d.%d.%d", sub[0], sub[1], sub[2], sub[3]+1)
	m["start"] = fmt.Sprintf("%d.%d.%d.%d", sub[0], sub[1], sub[2], sub[3]+2)
	m["end"] = fmt.Sprintf("%d.%d.%d.%d",
		sub[0]+(0xff-msk[0]), sub[1]+(0xff-msk[1]), sub[2]+(0xff-msk[2]), sub[3]+(0xff-msk[3]-1))

	return m, nil
}
