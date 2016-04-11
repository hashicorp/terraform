package cloudstack

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackVPNConnection() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackVPNConnectionCreate,
		Read:   resourceCloudStackVPNConnectionRead,
		Delete: resourceCloudStackVPNConnectionDelete,

		Schema: map[string]*schema.Schema{
			"customer_gateway_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"customergatewayid": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "Please use the `customer_gateway_id` field instead",
			},

			"vpn_gateway_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"vpngatewayid": &schema.Schema{
				Type:       schema.TypeString,
				Optional:   true,
				ForceNew:   true,
				Deprecated: "Please use the `vpn_gateway_id` field instead",
			},
		},
	}
}

func resourceCloudStackVPNConnectionCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	customergatewayid, ok := d.GetOk("customer_gateway_id")
	if !ok {
		customergatewayid, ok = d.GetOk("customergatewayid")
	}
	if !ok {
		return errors.New(
			"Either `customer_gateway_id` or [deprecated] `customergatewayid` must be provided.")
	}

	vpngatewayid, ok := d.GetOk("vpn_gateway_id")
	if !ok {
		vpngatewayid, ok = d.GetOk("vpngatewayid")
	}
	if !ok {
		return errors.New("Either `vpn_gateway_id` or [deprecated] `vpngatewayid` must be provided.")
	}

	// Create a new parameter struct
	p := cs.VPN.NewCreateVpnConnectionParams(
		customergatewayid.(string),
		vpngatewayid.(string),
	)

	// Create the new VPN Connection
	v, err := cs.VPN.CreateVpnConnection(p)
	if err != nil {
		return fmt.Errorf("Error creating VPN Connection: %s", err)
	}

	d.SetId(v.Id)

	return resourceCloudStackVPNConnectionRead(d, meta)
}

func resourceCloudStackVPNConnectionRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the VPN Connection details
	v, count, err := cs.VPN.GetVpnConnectionByID(d.Id())
	if err != nil {
		if count == 0 {
			log.Printf("[DEBUG] VPN Connection does no longer exist")
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("customer_gateway_id", v.S2scustomergatewayid)
	d.Set("vpn_gateway_id", v.S2svpngatewayid)

	return nil
}

func resourceCloudStackVPNConnectionDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.VPN.NewDeleteVpnConnectionParams(d.Id())

	// Delete the VPN Connection
	_, err := cs.VPN.DeleteVpnConnection(p)
	if err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error deleting VPN Connection: %s", err)
	}

	return nil
}
