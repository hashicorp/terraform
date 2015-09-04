package google

import (
	"fmt"
	"time"

	"github.com/hashicorp/terraform/helper/schema"

	"google.golang.org/api/compute/v1"
)

func resourceComputeVpnTunnel() *schema.Resource {
	return &schema.Resource{
		// Unfortunately, the VPNTunnelService does not support update
		// operations. This is why everything is marked forcenew
		Create: resourceComputeVpnTunnelCreate,
		Read:	resourceComputeVpnTunnelRead,
		Delete: resourceComputeVpnTunnelDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:	  schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"description": &schema.Schema{
				Type:	  schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"region": &schema.Schema{
				Type:	  schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"peer_ip": &schema.Schema{
				Type:	  schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"shared_secret": &schema.Schema{
				Type:	  schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"target_vpn_gateway": &schema.Schema{
				Type:	  schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"ike_version": &schema.Schema{
				Type:	  schema.TypeInt,
				Optional: true,
				Default:  2,
				ForceNew: true,
			},
			"detailed_status": &schema.Schema{
				Type:	  schema.TypeString,
				Computed: true,
			},
			"self_link": &schema.Schema{
				Type:	  schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceOperationWaitRegion(config *Config, op *compute.Operation, region, activity string) error {
	w := &OperationWaiter{
		Service: config.clientCompute,
		Op:		 op,
		Project: config.Project,
		Type:	 OperationWaitRegion,
		Region:  region,
	}

	state := w.Conf()
	state.Timeout = 2 * time.Minute
	state.MinTimeout = 1 * time.Second
	opRaw, err := state.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for %s: %s", activity, err)
	}

	op = opRaw.(*compute.Operation)
	if op.Error != nil {
		return OperationError(*op.Error)
	}

	return nil
}

func resourceComputeVpnTunnelCreate(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Get("name").(string)
	description := d.Get("description").(string)
	region := d.Get("region").(string)
	peerIp := d.Get("peer_ip").(string)
	sharedSecret  := d.Get("shared_secret").(string)
	targetVpnGateway := d.Get("target_vpn_gateway").(string)
	ikeVersion := d.Get("ike_version").(int)
	project := config.Project

	if ikeVersion < 1 || ikeVersion > 2 {
		return fmt.Errorf("Only IKE version 1 or 2 supported, not %d", ikeVersion)
	}

	vpnTunnelsService := compute.NewVpnTunnelsService(config.clientCompute)

	vpnTunnel := &compute.VpnTunnel{
		Name:			  name,
		Description:	  description,
		PeerIp:			  peerIp,
		SharedSecret:	  sharedSecret,
		TargetVpnGateway: targetVpnGateway,
		IkeVersion:		  int64(ikeVersion),
	}

	op, err := vpnTunnelsService.Insert(project, region, vpnTunnel).Do()
	if err != nil {
		return fmt.Errorf("Error Inserting VPN Tunnel %s : %s", name, err)
	}

	err = resourceOperationWaitRegion(config, op, region, "Inserting VPN Tunnel")
	if err != nil {
		return fmt.Errorf("Error Waiting to Insert VPN Tunnel %s: %s", name, err)
	}

	return resourceComputeVpnTunnelRead(d, meta);
}

func resourceComputeVpnTunnelRead(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Get("name").(string)
	region := d.Get("region").(string)
	project := config.Project

	vpnTunnelsService := compute.NewVpnTunnelsService(config.clientCompute)

	vpnTunnel, err := vpnTunnelsService.Get(project, region, name).Do()
	if err != nil {
		return fmt.Errorf("Error Reading VPN Tunnel %s: %s", name, err)
	}

	d.Set("detailed_status", vpnTunnel.DetailedStatus);
	d.Set("self_link", vpnTunnel.SelfLink);

	d.SetId(name)

	return nil
}

func resourceComputeVpnTunnelDelete(d *schema.ResourceData, meta interface{}) error {
	config := meta.(*Config)

	name := d.Get("name").(string)
	region := d.Get("region").(string)
	project := config.Project

	vpnTunnelsService := compute.NewVpnTunnelsService(config.clientCompute)

	op, err := vpnTunnelsService.Delete(project, region, name).Do()
	if err != nil {
		return fmt.Errorf("Error Reading VPN Tunnel %s: %s", name, err)
	}

	err = resourceOperationWaitRegion(config, op, region, "Deleting VPN Tunnel")
	if err != nil {
		return fmt.Errorf("Error Waiting to Delete VPN Tunnel %s: %s", name, err)
	}

	return nil
}
