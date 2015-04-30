package azure

import (
	"fmt"
	"log"
	"strings"

	"github.com/Azure/azure-sdk-for-go/management"
	"github.com/Azure/azure-sdk-for-go/management/virtualnetwork"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/mapstructure"
)

const (
	virtualNetworkRetrievalError = "Error retrieving Virtual Network Configuration: %s"
)

func resourceAzureVirtualNetwork() *schema.Resource {
	return &schema.Resource{
		Create: resourceAzureVirtualNetworkCreate,
		Read:   resourceAzureVirtualNetworkRead,
		Update: resourceAzureVirtualNetworkUpdate,
		Delete: resourceAzureVirtualNetworkDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"address_space": &schema.Schema{
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"subnet": &schema.Schema{
				Type:     schema.TypeMap,
				Required: true,
			},

			"location": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAzureVirtualNetworkCreate(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*management.Client)

	name := d.Get("name").(string)

	nc, err := virtualnetwork.NewClient(*mc).GetVirtualNetworkConfiguration()
	if err != nil {
		if strings.Contains(err.Error(), "ResourceNotFound") {
			nc = virtualnetwork.NetworkConfiguration{}
		} else {
			return fmt.Errorf(virtualNetworkRetrievalError, err)
		}
	}

	for _, n := range nc.Configuration.VirtualNetworkSites {
		if n.Name == name {
			return fmt.Errorf("Virtual Network %s already exists!", name)
		}
	}

	network, err := createVirtualNetwork(d)
	if err != nil {
		return err
	}

	nc.Configuration.VirtualNetworkSites = append(nc.Configuration.VirtualNetworkSites, network)

	err = virtualnetwork.NewClient(*mc).SetVirtualNetworkConfiguration(nc)
	if err != nil {
		return fmt.Errorf("Error creating Virtual Network %s: %s", name, err)
	}

	d.SetId(name)

	return resourceAzureVirtualNetworkRead(d, meta)
}

func resourceAzureVirtualNetworkRead(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*management.Client)

	nc, err := virtualnetwork.NewClient(*mc).GetVirtualNetworkConfiguration()
	if err != nil {
		return fmt.Errorf(virtualNetworkRetrievalError, err)
	}

	for _, n := range nc.Configuration.VirtualNetworkSites {
		if n.Name == d.Id() {
			d.Set("address_space", n.AddressSpace.AddressPrefix)
			d.Set("location", n.Location)

			subnets := map[string]interface{}{}
			for _, s := range n.Subnets {
				subnets[s.Name] = s.AddressPrefix
			}

			d.Set("subnet", subnets)

			return nil
		}
	}

	log.Printf("[DEBUG] Virtual Network %s does no longer exist", d.Id())
	d.SetId("")

	return nil
}

func resourceAzureVirtualNetworkUpdate(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*management.Client)

	nc, err := virtualnetwork.NewClient(*mc).GetVirtualNetworkConfiguration()
	if err != nil {
		return fmt.Errorf(virtualNetworkRetrievalError, err)
	}

	found := false
	for i, n := range nc.Configuration.VirtualNetworkSites {
		if n.Name == d.Id() {
			network, err := createVirtualNetwork(d)
			if err != nil {
				return err
			}

			nc.Configuration.VirtualNetworkSites[i] = network

			found = true
		}
	}

	if !found {
		return fmt.Errorf("Virtual Network %s does not exists!", d.Id())
	}

	err = virtualnetwork.NewClient(*mc).SetVirtualNetworkConfiguration(nc)
	if err != nil {
		return fmt.Errorf("Error updating Virtual Network %s: %s", d.Id(), err)
	}

	return resourceAzureVirtualNetworkRead(d, meta)
}

func resourceAzureVirtualNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*management.Client)

	nc, err := virtualnetwork.NewClient(*mc).GetVirtualNetworkConfiguration()
	if err != nil {
		return fmt.Errorf(virtualNetworkRetrievalError, err)
	}

	filtered := nc.Configuration.VirtualNetworkSites[:0]
	for _, n := range nc.Configuration.VirtualNetworkSites {
		if n.Name != d.Id() {
			filtered = append(filtered, n)
		}
	}

	nc.Configuration.VirtualNetworkSites = filtered

	err = virtualnetwork.NewClient(*mc).SetVirtualNetworkConfiguration(nc)
	if err != nil {
		return fmt.Errorf("Error deleting Virtual Network %s: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}

func createVirtualNetwork(d *schema.ResourceData) (virtualnetwork.VirtualNetworkSite, error) {
	var addressPrefix []string
	err := mapstructure.WeakDecode(d.Get("address_space"), &addressPrefix)
	if err != nil {
		return virtualnetwork.VirtualNetworkSite{}, fmt.Errorf("Error decoding address_space: %s", err)
	}

	addressSpace := virtualnetwork.AddressSpace{
		AddressPrefix: addressPrefix,
	}

	subnets := []virtualnetwork.Subnet{}
	for n, p := range d.Get("subnet").(map[string]interface{}) {
		subnets = append(subnets, virtualnetwork.Subnet{
			Name:          n,
			AddressPrefix: p.(string),
		})
	}

	return virtualnetwork.VirtualNetworkSite{
		Name:         d.Get("name").(string),
		Location:     d.Get("location").(string),
		AddressSpace: addressSpace,
		Subnets:      subnets,
	}, nil
}
