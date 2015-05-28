package azure

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/mapstructure"
	"github.com/svanharmelen/azure-sdk-for-go/management"
	"github.com/svanharmelen/azure-sdk-for-go/management/networksecuritygroup"
	"github.com/svanharmelen/azure-sdk-for-go/management/virtualnetwork"
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
				Type:     schema.TypeSet,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"address_prefix": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"security_group": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
				Set: resourceAzureSubnetHash,
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
	ac := meta.(*Client)
	mc := ac.mgmtClient

	name := d.Get("name").(string)

	// Lock the client just before we get the virtual network configuration and immediately
	// set an defer to unlock the client again whenever this function exits
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	nc, err := virtualnetwork.NewClient(mc).GetVirtualNetworkConfiguration()
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

	req, err := virtualnetwork.NewClient(mc).SetVirtualNetworkConfiguration(nc)
	if err != nil {
		return fmt.Errorf("Error creating Virtual Network %s: %s", name, err)
	}

	// Wait until the virtual network is created
	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf("Error waiting for Virtual Network %s to be created: %s", name, err)
	}

	d.SetId(name)

	if err := associateSecurityGroups(d, meta); err != nil {
		return err
	}

	return resourceAzureVirtualNetworkRead(d, meta)
}

func resourceAzureVirtualNetworkRead(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*Client).mgmtClient

	nc, err := virtualnetwork.NewClient(mc).GetVirtualNetworkConfiguration()
	if err != nil {
		return fmt.Errorf(virtualNetworkRetrievalError, err)
	}

	for _, n := range nc.Configuration.VirtualNetworkSites {
		if n.Name == d.Id() {
			d.Set("address_space", n.AddressSpace.AddressPrefix)
			d.Set("location", n.Location)

			// Create a new set to hold all configured subnets
			subnets := &schema.Set{
				F: resourceAzureSubnetHash,
			}

			// Loop through all endpoints
			for _, s := range n.Subnets {
				subnet := map[string]interface{}{}

				// Get the associated (if any) security group
				sg, err := networksecuritygroup.NewClient(mc).
					GetNetworkSecurityGroupForSubnet(s.Name, d.Id())
				if err != nil && !management.IsResourceNotFoundError(err) {
					return fmt.Errorf(
						"Error retrieving Network Security Group associations of subnet %s: %s", s.Name, err)
				}

				// Update the values
				subnet["name"] = s.Name
				subnet["address_prefix"] = s.AddressPrefix
				subnet["security_group"] = sg.Name

				subnets.Add(subnet)
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
	ac := meta.(*Client)
	mc := ac.mgmtClient

	// Lock the client just before we get the virtual network configuration and immediately
	// set an defer to unlock the client again whenever this function exits
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	nc, err := virtualnetwork.NewClient(mc).GetVirtualNetworkConfiguration()
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

	req, err := virtualnetwork.NewClient(mc).SetVirtualNetworkConfiguration(nc)
	if err != nil {
		return fmt.Errorf("Error updating Virtual Network %s: %s", d.Id(), err)
	}

	// Wait until the virtual network is updated
	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf("Error waiting for Virtual Network %s to be updated: %s", d.Id(), err)
	}

	if err := associateSecurityGroups(d, meta); err != nil {
		return err
	}

	return resourceAzureVirtualNetworkRead(d, meta)
}

func resourceAzureVirtualNetworkDelete(d *schema.ResourceData, meta interface{}) error {
	ac := meta.(*Client)
	mc := ac.mgmtClient

	// Lock the client just before we get the virtual network configuration and immediately
	// set an defer to unlock the client again whenever this function exits
	ac.mutex.Lock()
	defer ac.mutex.Unlock()

	nc, err := virtualnetwork.NewClient(mc).GetVirtualNetworkConfiguration()
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

	req, err := virtualnetwork.NewClient(mc).SetVirtualNetworkConfiguration(nc)
	if err != nil {
		return fmt.Errorf("Error deleting Virtual Network %s: %s", d.Id(), err)
	}

	// Wait until the virtual network is deleted
	if err := mc.WaitForOperation(req, nil); err != nil {
		return fmt.Errorf("Error waiting for Virtual Network %s to be deleted: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}

func resourceAzureSubnetHash(v interface{}) int {
	m := v.(map[string]interface{})
	subnet := m["name"].(string) + m["address_prefix"].(string) + m["security_group"].(string)
	return hashcode.String(subnet)
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

	// Add all subnets that are configured
	var subnets []virtualnetwork.Subnet
	if rs := d.Get("subnet").(*schema.Set); rs.Len() > 0 {
		for _, subnet := range rs.List() {
			subnet := subnet.(map[string]interface{})
			subnets = append(subnets, virtualnetwork.Subnet{
				Name:          subnet["name"].(string),
				AddressPrefix: subnet["address_prefix"].(string),
			})
		}
	}

	return virtualnetwork.VirtualNetworkSite{
		Name:         d.Get("name").(string),
		Location:     d.Get("location").(string),
		AddressSpace: addressSpace,
		Subnets:      subnets,
	}, nil
}

func associateSecurityGroups(d *schema.ResourceData, meta interface{}) error {
	mc := meta.(*Client).mgmtClient
	nsgClient := networksecuritygroup.NewClient(mc)

	virtualNetwork := d.Get("name").(string)

	if rs := d.Get("subnet").(*schema.Set); rs.Len() > 0 {
		for _, subnet := range rs.List() {
			subnet := subnet.(map[string]interface{})
			securityGroup := subnet["security_group"].(string)
			subnetName := subnet["name"].(string)

			// Get the associated (if any) security group
			sg, err := nsgClient.GetNetworkSecurityGroupForSubnet(subnetName, d.Id())
			if err != nil && !management.IsResourceNotFoundError(err) {
				return fmt.Errorf(
					"Error retrieving Network Security Group associations of subnet %s: %s", subnetName, err)
			}

			// If the desired and actual security group are the same, were done so can just continue
			if sg.Name == securityGroup {
				continue
			}

			// If there is an associated security group, make sure we first remove it from the subnet
			if sg.Name != "" {
				req, err := nsgClient.RemoveNetworkSecurityGroupFromSubnet(sg.Name, subnetName, virtualNetwork)
				if err != nil {
					return fmt.Errorf("Error removing Network Security Group %s from subnet %s: %s",
						securityGroup, subnetName, err)
				}

				// Wait until the security group is associated
				if err := mc.WaitForOperation(req, nil); err != nil {
					return fmt.Errorf(
						"Error waiting for Network Security Group %s to be removed from subnet %s: %s",
						securityGroup, subnetName, err)
				}
			}

			// If the desired security group is not empty, assign the security group to the subnet
			if securityGroup != "" {
				req, err := nsgClient.AddNetworkSecurityToSubnet(securityGroup, subnetName, virtualNetwork)
				if err != nil {
					return fmt.Errorf("Error associating Network Security Group %s to subnet %s: %s",
						securityGroup, subnetName, err)
				}

				// Wait until the security group is associated
				if err := mc.WaitForOperation(req, nil); err != nil {
					return fmt.Errorf(
						"Error waiting for Network Security Group %s to be associated with subnet %s: %s",
						securityGroup, subnetName, err)
				}
			}

		}
	}

	return nil
}
