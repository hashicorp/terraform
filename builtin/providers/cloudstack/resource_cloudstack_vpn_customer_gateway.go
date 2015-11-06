package cloudstack

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/xanzy/go-cloudstack/cloudstack"
)

func resourceCloudStackVPNCustomerGateway() *schema.Resource {
	return &schema.Resource{
		Create: resourceCloudStackVPNCustomerGatewayCreate,
		Read:   resourceCloudStackVPNCustomerGatewayRead,
		Update: resourceCloudStackVPNCustomerGatewayUpdate,
		Delete: resourceCloudStackVPNCustomerGatewayDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"cidr": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"esp_policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"gateway": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"ike_policy": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"ipsec_psk": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"dpd": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"esp_lifetime": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"ike_lifetime": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceCloudStackVPNCustomerGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.VPN.NewCreateVpnCustomerGatewayParams(
		d.Get("cidr").(string),
		d.Get("esp_policy").(string),
		d.Get("gateway").(string),
		d.Get("ike_policy").(string),
		d.Get("ipsec_psk").(string),
	)

	p.SetName(d.Get("name").(string))

	if dpd, ok := d.GetOk("dpd"); ok {
		p.SetDpd(dpd.(bool))
	}

	if esplifetime, ok := d.GetOk("esp_lifetime"); ok {
		p.SetEsplifetime(int64(esplifetime.(int)))
	}

	if ikelifetime, ok := d.GetOk("ike_lifetime"); ok {
		p.SetIkelifetime(int64(ikelifetime.(int)))
	}

	// Create the new VPN Customer Gateway
	v, err := cs.VPN.CreateVpnCustomerGateway(p)
	if err != nil {
		return fmt.Errorf("Error creating VPN Customer Gateway %s: %s", d.Get("name").(string), err)
	}

	d.SetId(v.Id)

	return resourceCloudStackVPNCustomerGatewayRead(d, meta)
}

func resourceCloudStackVPNCustomerGatewayRead(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Get the VPN Customer Gateway details
	v, count, err := cs.VPN.GetVpnCustomerGatewayByID(d.Id())
	if err != nil {
		if count == 0 {
			log.Printf(
				"[DEBUG] VPN Customer Gateway %s does no longer exist", d.Get("name").(string))
			d.SetId("")
			return nil
		}

		return err
	}

	d.Set("name", v.Name)
	d.Set("cidr", v.Cidrlist)
	d.Set("esp_policy", v.Esppolicy)
	d.Set("gateway", v.Gateway)
	d.Set("ike_policy", v.Ikepolicy)
	d.Set("ipsec_psk", v.Ipsecpsk)
	d.Set("dpd", v.Dpd)
	d.Set("esp_lifetime", int(v.Esplifetime))
	d.Set("ike_lifetime", int(v.Ikelifetime))

	return nil
}

func resourceCloudStackVPNCustomerGatewayUpdate(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.VPN.NewUpdateVpnCustomerGatewayParams(
		d.Get("cidr").(string),
		d.Get("esp_policy").(string),
		d.Get("gateway").(string),
		d.Id(),
		d.Get("ike_policy").(string),
		d.Get("ipsec_psk").(string),
	)

	p.SetName(d.Get("name").(string))

	if dpd, ok := d.GetOk("dpd"); ok {
		p.SetDpd(dpd.(bool))
	}

	if esplifetime, ok := d.GetOk("esp_lifetime"); ok {
		p.SetEsplifetime(int64(esplifetime.(int)))
	}

	if ikelifetime, ok := d.GetOk("ike_lifetime"); ok {
		p.SetIkelifetime(int64(ikelifetime.(int)))
	}

	// Update the VPN Customer Gateway
	_, err := cs.VPN.UpdateVpnCustomerGateway(p)
	if err != nil {
		return fmt.Errorf("Error updating VPN Customer Gateway %s: %s", d.Get("name").(string), err)
	}

	return resourceCloudStackVPNCustomerGatewayRead(d, meta)
}

func resourceCloudStackVPNCustomerGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	cs := meta.(*cloudstack.CloudStackClient)

	// Create a new parameter struct
	p := cs.VPN.NewDeleteVpnCustomerGatewayParams(d.Id())

	// Delete the VPN Customer Gateway
	_, err := cs.VPN.DeleteVpnCustomerGateway(p)
	if err != nil {
		// This is a very poor way to be told the ID does no longer exist :(
		if strings.Contains(err.Error(), fmt.Sprintf(
			"Invalid parameter id value=%s due to incorrect long value format, "+
				"or entity does not exist", d.Id())) {
			return nil
		}

		return fmt.Errorf("Error deleting VPN Customer Gateway %s: %s", d.Get("name").(string), err)
	}

	return nil
}
