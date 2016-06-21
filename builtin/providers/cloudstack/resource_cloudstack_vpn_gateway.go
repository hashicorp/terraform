package cloudstack

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackVPNGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackVPNGatewayCreate,
		Read:   resourceCloudStackVPNGatewayRead,
		Delete: resourceCloudStackVPNGatewayDelete,

		Schema: map[string]*schema.Schema{
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

			"public_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceCloudStackVPNGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	vpc, ok := d.GetOk("vpc_id")
	if !ok {
		vpc, ok = d.GetOk("vpc")
	}
	if !ok {
		return errors.New("Either `vpc_id` or [deprecated] `vpc` must be provided.")
	}

	// Retrieve the VPC ID
	vpcid, e := retrieveID(cs, "vpc", vpc.(string))
	if e != nil {
		return e.Error()
	}

	// Create a new parameter struct
	p := cs.VPN.NewCreateVpnGatewayParams(vpcid)

	// Create the new VPN Gateway
	v, err := cs.VPN.CreateVpnGateway(p)
	if err != nil {
		return fmt.Errorf("Error creating VPN Gateway for VPC ID %s: %s", vpcid, err)
	}

	d.SetId(v.Id)

	return resourceCloudStackVPNGatewayRead(d, meta)
}

func resourceCloudStackVPNGatewayRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the VPN Gateway details
	v, count, err := cs.VPN.GetVpnGatewayByID(d.Id())
	if err != nil {
		if count == 0 {
			log.Printf(
				"[DEBUG] VPN Gateway for VPC ID %s does no longer exist", d.Get("vpc_id").(string))
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("vpc_id", v.Vpcid)
	d.Set("public_ip", v.Publicip)

	return nil
}

func resourceCloudStackVPNGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.VPN.NewDeleteVpnGatewayParams(d.Id())

	// Delete the VPN Gateway
	_, err := cs.VPN.DeleteVpnGateway(p)
	if err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error deleting VPN Gateway for VPC %s: %s", d.Get("vpc_id").(string), err)
	}

	return nil
}
