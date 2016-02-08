package azurerm

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmSubnet() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmSubnetCreate,
		Read:   resourceArmSubnetRead,
		Update: resourceArmSubnetCreate,
		Delete: resourceArmSubnetDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"resource_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"virtual_network_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"address_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"network_security_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"route_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"ip_configurations": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceArmSubnetCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	subnetClient := client.subnetClient

	log.Printf("[INFO] preparing arguments for Azure ARM Subnet creation.")

	name := d.Get("name").(string)
	vnetName := d.Get("virtual_network_name").(string)
	resGroup := d.Get("resource_group_name").(string)
	addressPrefix := d.Get("address_prefix").(string)

	armMutexKV.Lock(vnetName)
	defer armMutexKV.Unlock(vnetName)

	properties := network.SubnetPropertiesFormat{
		AddressPrefix: &addressPrefix,
	}

	if v, ok := d.GetOk("network_security_group_id"); ok {
		nsgId := v.(string)
		properties.NetworkSecurityGroup = &network.SecurityGroup{
			ID: &nsgId,
		}
	}

	if v, ok := d.GetOk("route_table_id"); ok {
		rtId := v.(string)
		properties.RouteTable = &network.RouteTable{
			ID: &rtId,
		}
	}

	subnet := network.Subnet{
		Name:       &name,
		Properties: &properties,
	}

	resp, err := subnetClient.CreateOrUpdate(resGroup, vnetName, name, subnet)
	if err != nil {
		return err
	}

	d.SetId(*resp.ID)

	log.Printf("[DEBUG] Waiting for Subnet (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  []string{"Succeeded"},
		Refresh: subnetRuleStateRefreshFunc(client, resGroup, vnetName, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Subnet (%s) to become available: %s", name, err)
	}

	return resourceArmSubnetRead(d, meta)
}

func resourceArmSubnetRead(d *schema.ResourceData, meta interface{}) error {
	subnetClient := meta.(*ArmClient).subnetClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	vnetName := id.Path["virtualNetworks"]
	name := id.Path["subnets"]

	resp, err := subnetClient.Get(resGroup, vnetName, name, "")
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Subnet %s: %s", name, err)
	}

	if resp.Properties.IPConfigurations != nil && len(*resp.Properties.IPConfigurations) > 0 {
		ips := make([]string, 0, len(*resp.Properties.IPConfigurations))
		for _, ip := range *resp.Properties.IPConfigurations {
			ips = append(ips, *ip.ID)
		}

		if err := d.Set("ip_configurations", ips); err != nil {
			return err
		}
	}

	return nil
}

func resourceArmSubnetDelete(d *schema.ResourceData, meta interface{}) error {
	subnetClient := meta.(*ArmClient).subnetClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["subnets"]
	vnetName := id.Path["virtualNetworks"]

	armMutexKV.Lock(vnetName)
	defer armMutexKV.Unlock(vnetName)

	_, err = subnetClient.Delete(resGroup, vnetName, name)

	return err
}

func subnetRuleStateRefreshFunc(client *ArmClient, resourceGroupName string, virtualNetworkName string, subnetName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.subnetClient.Get(resourceGroupName, virtualNetworkName, subnetName, "")
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in subnetRuleStateRefreshFunc to Azure ARM for subnet '%s' (RG: '%s') (VNN: '%s'): %s", subnetName, resourceGroupName, virtualNetworkName, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}
