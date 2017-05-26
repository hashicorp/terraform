package cloudstack

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

const none = "none"

func resourceCloudStackNetwork() *schema.Resource {
	aclidSchema := &schema.Schema{
		Type:     schema.TypeString,
		Optional: true,
		Default:  none,
	}

	aclidSchema.StateFunc = func(v interface{}) string {
		value := v.(string)

		if value == none {
			aclidSchema.ForceNew = true
		} else {
			aclidSchema.ForceNew = false
		}

		return value
	}

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

			"gateway": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"startip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"endip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"network_domain": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"network_offering": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"vlan": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"acl_id": aclidSchema,

			"project": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"tags": tagsSchema(),
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

	// Get the network offering to check if it supports specifying IP ranges
	no, _, err := cs.NetworkOffering.GetNetworkOfferingByID(networkofferingid)
	if err != nil {
		return err
	}

	m, err := parseCIDR(d, no.Specifyipranges)
	if err != nil {
		return err
	}

	// Set the needed IP config
	p.SetGateway(m["gateway"])
	p.SetNetmask(m["netmask"])

	// Only set the start IP if we have one
	if startip, ok := m["startip"]; ok {
		p.SetStartip(startip)
	}

	// Only set the end IP if we have one
	if endip, ok := m["endip"]; ok {
		p.SetEndip(endip)
	}

	// Set the network domain if we have one
	if networkDomain, ok := d.GetOk("network_domain"); ok {
		p.SetNetworkdomain(networkDomain.(string))
	}

	if vlan, ok := d.GetOk("vlan"); ok {
		p.SetVlan(strconv.Itoa(vlan.(int)))
	}

	// Check is this network needs to be created in a VPC
	if vpcid, ok := d.GetOk("vpc_id"); ok {
		// Set the vpc id
		p.SetVpcid(vpcid.(string))

		// Since we're in a VPC, check if we want to assiciate an ACL list
		if aclid, ok := d.GetOk("acl_id"); ok && aclid.(string) != none {
			// Set the acl ID
			p.SetAclid(aclid.(string))
		}
	}

	// If there is a project supplied, we retrieve and set the project id
	if err := setProjectid(p, cs, d); err != nil {
		return err
	}

	// Create the new network
	r, err := cs.Network.CreateNetwork(p)
	if err != nil {
		return fmt.Errorf("Error creating network %s: %s", name, err)
	}

	d.SetId(r.Id)

	err = setTags(cs, d, "network")
	if err != nil {
		return fmt.Errorf("Error setting tags: %s", err)
	}

	return resourceCloudStackNetworkRead(d, meta)
}

func resourceCloudStackNetworkRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the virtual machine details
	n, count, err := cs.Network.GetNetworkByID(
		d.Id(),
		cloudstack.WithProject(d.Get("project").(string)),
	)
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
	d.Set("gateway", n.Gateway)
	d.Set("network_domain", n.Networkdomain)
	d.Set("vpc_id", n.Vpcid)

	if n.Aclid == "" {
		n.Aclid = none
	}
	d.Set("acl_id", n.Aclid)

	// Read the tags and store them in a map
	tags := make(map[string]interface{})
	for item := range n.Tags {
		tags[n.Tags[item].Key] = n.Tags[item].Value
	}
	d.Set("tags", tags)

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

	// Check if the network domain is changed
	if d.HasChange("network_domain") {
		p.SetNetworkdomain(d.Get("network_domain").(string))
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

	// Replace the ACL if the ID has changed
	if d.HasChange("acl_id") {
		p := cs.NetworkACL.NewReplaceNetworkACLListParams(d.Get("acl_id").(string))
		p.SetNetworkid(d.Id())

		_, err := cs.NetworkACL.ReplaceNetworkACLList(p)
		if err != nil {
			return fmt.Errorf("Error replacing ACL: %s", err)
		}
	}

	// Update tags if they have changed
	if d.HasChange("tags") {
		err = setTags(cs, d, "network")
		if err != nil {
			return fmt.Errorf("Error updating tags: %s", err)
		}
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

func parseCIDR(d *schema.ResourceData, specifyiprange bool) (map[string]string, error) {
	m := make(map[string]string, 4)

	cidr := d.Get("cidr").(string)
	ip, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse cidr %s: %s", cidr, err)
	}

	msk := ipnet.Mask
	sub := ip.Mask(msk)

	m["netmask"] = fmt.Sprintf("%d.%d.%d.%d", msk[0], msk[1], msk[2], msk[3])

	if gateway, ok := d.GetOk("gateway"); ok {
		m["gateway"] = gateway.(string)
	} else {
		m["gateway"] = fmt.Sprintf("%d.%d.%d.%d", sub[0], sub[1], sub[2], sub[3]+1)
	}

	if startip, ok := d.GetOk("startip"); ok {
		m["startip"] = startip.(string)
	} else if specifyiprange {
		m["startip"] = fmt.Sprintf("%d.%d.%d.%d", sub[0], sub[1], sub[2], sub[3]+2)
	}

	if endip, ok := d.GetOk("endip"); ok {
		m["endip"] = endip.(string)
	} else if specifyiprange {
		m["endip"] = fmt.Sprintf("%d.%d.%d.%d",
			sub[0]+(0xff-msk[0]), sub[1]+(0xff-msk[1]), sub[2]+(0xff-msk[2]), sub[3]+(0xff-msk[3]-1))
	}

	return m, nil
}
