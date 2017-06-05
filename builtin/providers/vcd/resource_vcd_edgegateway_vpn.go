package vcd

import (
	"fmt"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	types "github.com/ukcloud/govcloudair/types/v56"
	"log"
)

func resourceVcdEdgeGatewayVpn() *schema.Resource {
	return &schema.Resource{
		Create: resourceVcdEdgeGatewayVpnCreate,
		Read:   resourceVcdEdgeGatewayVpnRead,
		Delete: resourceVcdEdgeGatewayVpnDelete,

		Schema: map[string]*schema.Schema{

			"edge_gateway": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

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

			"encryption_protocol": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"local_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"local_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"mtu": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"peer_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"peer_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"shared_secret": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"local_subnets": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"local_subnet_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"local_subnet_gateway": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"local_subnet_mask": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},

			"peer_subnets": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"peer_subnet_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"peer_subnet_gateway": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"peer_subnet_mask": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceVcdEdgeGatewayVpnCreate(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)
	log.Printf("[TRACE] CLIENT: %#v", vcdClient)
	vcdClient.Mutex.Lock()
	defer vcdClient.Mutex.Unlock()

	edgeGateway, err := vcdClient.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))

	localSubnetsList := d.Get("local_subnets").(*schema.Set).List()
	peerSubnetsList := d.Get("peer_subnets").(*schema.Set).List()

	localSubnets := make([]*types.IpsecVpnSubnet, len(localSubnetsList))
	peerSubnets := make([]*types.IpsecVpnSubnet, len(peerSubnetsList))

	for i, s := range localSubnetsList {
		ls := s.(map[string]interface{})
		localSubnets[i] = &types.IpsecVpnSubnet{
			Name:    ls["local_subnet_name"].(string),
			Gateway: ls["local_subnet_gateway"].(string),
			Netmask: ls["local_subnet_mask"].(string),
		}
	}

	for i, s := range peerSubnetsList {
		ls := s.(map[string]interface{})
		peerSubnets[i] = &types.IpsecVpnSubnet{
			Name:    ls["peer_subnet_name"].(string),
			Gateway: ls["peer_subnet_gateway"].(string),
			Netmask: ls["peer_subnet_mask"].(string),
		}
	}

	tunnel := &types.GatewayIpsecVpnTunnel{
		Name:        d.Get("name").(string),
		Description: d.Get("description").(string),
		IpsecVpnLocalPeer: &types.IpsecVpnLocalPeer{
			ID:   "",
			Name: "",
		},
		EncryptionProtocol: d.Get("encryption_protocol").(string),
		LocalIPAddress:     d.Get("local_ip_address").(string),
		LocalID:            d.Get("local_id").(string),
		LocalSubnet:        localSubnets,
		Mtu:                d.Get("mtu").(int),
		PeerID:             d.Get("peer_id").(string),
		PeerIPAddress:      d.Get("peer_ip_address").(string),
		PeerSubnet:         peerSubnets,
		SharedSecret:       d.Get("shared_secret").(string),
		IsEnabled:          true,
	}

	tunnels := make([]*types.GatewayIpsecVpnTunnel, 1)
	tunnels[0] = tunnel

	ipsecVPNConfig := &types.EdgeGatewayServiceConfiguration{
		Xmlns: "http://www.vmware.com/vcloud/v1.5",
		GatewayIpsecVpnService: &types.GatewayIpsecVpnService{
			IsEnabled: true,
			Tunnel:    tunnels,
		},
	}

	log.Printf("[INFO] ipsecVPNConfig: %#v", ipsecVPNConfig)

	err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		edgeGateway.Refresh()
		task, err := edgeGateway.AddIpsecVPN(ipsecVPNConfig)
		if err != nil {
			log.Printf("[INFO] Error setting ipsecVPNConfig rules: %s", err)
			return resource.RetryableError(
				fmt.Errorf("Error setting ipsecVPNConfig rules: %#v", err))
		}

		return resource.RetryableError(task.WaitTaskCompletion())
	})
	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}

	d.SetId(d.Get("edge_gateway").(string))

	return resourceVcdEdgeGatewayVpnRead(d, meta)
}

func resourceVcdEdgeGatewayVpnDelete(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	log.Printf("[TRACE] CLIENT: %#v", vcdClient)

	vcdClient.Mutex.Lock()
	defer vcdClient.Mutex.Unlock()

	edgeGateway, err := vcdClient.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))

	ipsecVPNConfig := &types.EdgeGatewayServiceConfiguration{
		Xmlns: "http://www.vmware.com/vcloud/v1.5",
		GatewayIpsecVpnService: &types.GatewayIpsecVpnService{
			IsEnabled: false,
		},
	}

	log.Printf("[INFO] ipsecVPNConfig: %#v", ipsecVPNConfig)

	err = retryCall(vcdClient.MaxRetryTimeout, func() *resource.RetryError {
		edgeGateway.Refresh()
		task, err := edgeGateway.AddIpsecVPN(ipsecVPNConfig)
		if err != nil {
			log.Printf("[INFO] Error setting ipsecVPNConfig rules: %s", err)
			return resource.RetryableError(
				fmt.Errorf("Error setting ipsecVPNConfig rules: %#v", err))
		}

		return resource.RetryableError(task.WaitTaskCompletion())
	})
	if err != nil {
		return fmt.Errorf("Error completing tasks: %#v", err)
	}

	d.SetId(d.Get("edge_gateway").(string))

	if err != nil {
		return fmt.Errorf("Error finding edge gateway: %#v", err)
	}

	return nil
}

func resourceVcdEdgeGatewayVpnRead(d *schema.ResourceData, meta interface{}) error {
	vcdClient := meta.(*VCDClient)

	edgeGateway, err := vcdClient.OrgVdc.FindEdgeGateway(d.Get("edge_gateway").(string))
	if err != nil {
		return fmt.Errorf("Error finding edge gateway: %#v", err)
	}

	egsc := edgeGateway.EdgeGateway.Configuration.EdgeGatewayServiceConfiguration.GatewayIpsecVpnService

	if len(egsc.Tunnel) == 0 {
		d.SetId("")
		return nil
	}

	if len(egsc.Tunnel) == 1 {
		tunnel := egsc.Tunnel[0]
		d.Set("name", tunnel.Name)
		d.Set("description", tunnel.Description)
		d.Set("encryption_protocol", tunnel.EncryptionProtocol)
		d.Set("local_ip_address", tunnel.LocalIPAddress)
		d.Set("local_id", tunnel.LocalID)
		d.Set("mtu", tunnel.Mtu)
		d.Set("peer_ip_address", tunnel.PeerIPAddress)
		d.Set("peer_id", tunnel.PeerID)
		d.Set("shared_secret", tunnel.SharedSecret)
		d.Set("local_subnets", tunnel.LocalSubnet)
		d.Set("peer_subnets", tunnel.PeerSubnet)
	} else {
		return fmt.Errorf("Multiple tunnels not currently supported")
	}

	return nil
}
