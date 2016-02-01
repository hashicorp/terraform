package google

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
)

func resourceComputeVpnGateway() *schema.Resource {
	return &schema.Resource{
		// Unfortunately, the VPNGatewayService does not support update
		// operations. This is why everything is marked forcenew
		Create: resourceComputeVpnGatewayCreate,
		Read:   resourceComputeVpnGatewayRead,
		Delete: resourceComputeVpnGatewayDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"network": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"region": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"self_link": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceComputeVpnGatewayCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Get("name").(string)
	network := d.Get("network").(string)
	region := getOptionalRegion(d, config)
	project := config.Project

	vpnGatewaysService := compute.NewTargetVpnGatewaysService(config.clientCompute)

	vpnGateway := &compute.TargetVpnGateway{
		Name:    name,
		Network: network,
	}

	if v, ok := d.GetOk("description"); ok {
		vpnGateway.Description = v.(string)
	}

	op, err := vpnGatewaysService.Insert(project, region, vpnGateway).Do()
	if err != nil {
		return fmt.Errorf("Error Inserting VPN Gateway %s into network %s: %s", name, network, err)
	}

	err = computeOperationWaitRegion(config, op, region, "Inserting VPN Gateway")
	if err != nil {
		return fmt.Errorf("Error Waiting to Insert VPN Gateway %s into network %s: %s", name, network, err)
	}

	return resourceComputeVpnGatewayRead(d, meta)
}

func resourceComputeVpnGatewayRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Get("name").(string)
	region := d.Get("region").(string)
	project := config.Project

	vpnGatewaysService := compute.NewTargetVpnGatewaysService(config.clientCompute)
	vpnGateway, err := vpnGatewaysService.Get(project, region, name).Do()

	if err != nil {
		if gerr, ok := err.(*googleapi.Error); ok && gerr.Code == 404 {
			log.Printf("[WARN] Removing VPN Gateway %q because it's gone", d.Get("name").(string))
			// The resource doesn't exist anymore
			d.SetId("")

			return nil
		}

		return fmt.Errorf("Error Reading VPN Gateway %s: %s", name, err)
	}

	d.Set("self_link", vpnGateway.SelfLink)
	d.SetId(name)

	return nil
}

func resourceComputeVpnGatewayDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Get("name").(string)
	region := d.Get("region").(string)
	project := config.Project

	vpnGatewaysService := compute.NewTargetVpnGatewaysService(config.clientCompute)

	op, err := vpnGatewaysService.Delete(project, region, name).Do()
	if err != nil {
		return fmt.Errorf("Error Reading VPN Gateway %s: %s", name, err)
	}

	err = computeOperationWaitRegion(config, op, region, "Deleting VPN Gateway")
	if err != nil {
		return fmt.Errorf("Error Waiting to Delete VPN Gateway %s: %s", name, err)
	}

	return nil
}
