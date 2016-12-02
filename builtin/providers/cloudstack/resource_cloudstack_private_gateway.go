package cloudstack

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackPrivateGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackPrivateGatewayCreate,
		Read:   resourceCloudStackPrivateGatewayRead,
		Update: resourceCloudStackPrivateGatewayUpdate,
		Delete: resourceCloudStackPrivateGatewayDelete,

		Schema: map[string]*schema.Schema{
			"gateway": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"netmask": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"vlan": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"physical_network_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"network_offering": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"acl_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"vpc_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceCloudStackPrivateGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	ipaddress := d.Get("ip_address").(string)
	networkofferingid := d.Get("network_offering").(string)

	// Create a new parameter struct
	p := cs.VPC.NewCreatePrivateGatewayParams(
		d.Get("gateway").(string),
		ipaddress,
		d.Get("netmask").(string),
		d.Get("vlan").(string),
		d.Get("vpc_id").(string),
	)

	// Retrieve the network_offering ID
	if networkofferingid != "" {
		networkofferingid, e := retrieveID(cs, "network_offering", networkofferingid)
		if e != nil {
			return e.Error()
		}
		p.SetNetworkofferingid(networkofferingid)
	}

	// Check if we want to associate an ACL
	if aclid, ok := d.GetOk("acl_id"); ok {
		// Set the acl ID
		p.SetAclid(aclid.(string))
	}

	// Create the new private gateway
	r, err := cs.VPC.CreatePrivateGateway(p)
	if err != nil {
		return fmt.Errorf("Error creating private gateway for %s: %s", ipaddress, err)
	}

	d.SetId(r.Id)

	return resourceCloudStackPrivateGatewayRead(d, meta)
}

func resourceCloudStackPrivateGatewayRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the private gateway details
	gw, count, err := cs.VPC.GetPrivateGatewayByID(d.Id())
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] Private gateway %s does no longer exist", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("gateway", gw.Gateway)
	d.Set("ip_address", gw.Ipaddress)
	d.Set("netmask", gw.Netmask)
	d.Set("vlan", gw.Vlan)
	d.Set("acl_id", gw.Aclid)
	d.Set("vpc_id", gw.Vpcid)

	return nil
}

func resourceCloudStackPrivateGatewayUpdate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Replace the ACL if the ID has changed
	if d.HasChange("acl_id") {
		p := cs.NetworkACL.NewReplaceNetworkACLListParams(d.Get("acl_id").(string))
		p.SetNetworkid(d.Id())

		_, err := cs.NetworkACL.ReplaceNetworkACLList(p)
		if err != nil {
			return fmt.Errorf("Error replacing ACL: %s", err)
		}
	}

	return resourceCloudStackNetworkRead(d, meta)
}

func resourceCloudStackPrivateGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.VPC.NewDeletePrivateGatewayParams(d.Id())

	// Delete the private gateway
	_, err := cs.VPC.DeletePrivateGateway(p)
	if err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error deleting private gateway %s: %s", d.Id(), err)
	}

	return nil
}
