package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/schema"
)

// peerMutex is used to prevet multiple Peering resources being creaed, updated
// or deleted at the same time
var peerMutex = &sync.Mutex{}

func resourceArmVirtualNetworkPeering() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmVirtualNetworkPeeringCreate,
		Read:   resourceArmVirtualNetworkPeeringRead,
		Update: resourceArmVirtualNetworkPeeringCreate,
		Delete: resourceArmVirtualNetworkPeeringDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"virtual_network_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"remote_virtual_network_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"allow_virtual_network_access": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"allow_forwarded_traffic": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"allow_gateway_transit": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"use_remote_gateways": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceArmVirtualNetworkPeeringCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).vnetPeeringsClient

	log.Printf("[INFO] preparing arguments for Azure ARM virtual network peering creation.")

	name := d.Get("name").(string)
	vnetName := d.Get("virtual_network_name").(string)
	resGroup := d.Get("resource_group_name").(string)

	peer := network.VirtualNetworkPeering{
		Name: &name,
		VirtualNetworkPeeringPropertiesFormat: getVirtualNetworkPeeringProperties(d),
	}

	peerMutex.Lock()
	defer peerMutex.Unlock()

	_, error := client.CreateOrUpdate(resGroup, vnetName, name, peer, make(chan struct{}))
	err := <-error
	if err != nil {
		return err
	}

	read, err := client.Get(resGroup, vnetName, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Virtual Network Peering %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmVirtualNetworkPeeringRead(d, meta)
}

func resourceArmVirtualNetworkPeeringRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).vnetPeeringsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	vnetName := id.Path["virtualNetworks"]
	name := id.Path["virtualNetworkPeerings"]

	resp, err := client.Get(resGroup, vnetName, name)
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure virtual network peering %s: %s", name, err)
	}

	peer := *resp.VirtualNetworkPeeringPropertiesFormat

	// update appropriate values
	d.Set("resource_group_name", resGroup)
	d.Set("name", resp.Name)
	d.Set("virtual_network_name", vnetName)
	d.Set("allow_virtual_network_access", peer.AllowVirtualNetworkAccess)
	d.Set("allow_forwarded_traffic", peer.AllowForwardedTraffic)
	d.Set("allow_gateway_transit", peer.AllowGatewayTransit)
	d.Set("use_remote_gateways", peer.UseRemoteGateways)
	d.Set("remote_virtual_network_id", peer.RemoteVirtualNetwork.ID)

	return nil
}

func resourceArmVirtualNetworkPeeringDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient).vnetPeeringsClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	vnetName := id.Path["virtualNetworks"]
	name := id.Path["virtualNetworkPeerings"]

	peerMutex.Lock()
	defer peerMutex.Unlock()

	_, error := client.Delete(resGroup, vnetName, name, make(chan struct{}))
	err = <-error

	return err
}

func getVirtualNetworkPeeringProperties(d *schema.ResourceData) *network.VirtualNetworkPeeringPropertiesFormat {
	allowVirtualNetworkAccess := d.Get("allow_virtual_network_access").(bool)
	allowForwardedTraffic := d.Get("allow_forwarded_traffic").(bool)
	allowGatewayTransit := d.Get("allow_gateway_transit").(bool)
	useRemoteGateways := d.Get("use_remote_gateways").(bool)
	remoteVirtualNetworkID := d.Get("remote_virtual_network_id").(string)

	return &network.VirtualNetworkPeeringPropertiesFormat{
		AllowVirtualNetworkAccess: &allowVirtualNetworkAccess,
		AllowForwardedTraffic:     &allowForwardedTraffic,
		AllowGatewayTransit:       &allowGatewayTransit,
		UseRemoteGateways:         &useRemoteGateways,
		RemoteVirtualNetwork: &network.SubResource{
			ID: &remoteVirtualNetworkID,
		},
	}
}
