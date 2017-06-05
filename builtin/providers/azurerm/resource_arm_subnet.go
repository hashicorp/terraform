package azurerm

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmSubnet() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmSubnetCreate,
		Read:   resourceArmSubnetRead,
		Update: resourceArmSubnetCreate,
		Delete: resourceArmSubnetDelete,
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

			"address_prefix": {
				Type:     schema.TypeString,
				Required: true,
			},

			"network_security_group_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"route_table_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"ip_configurations": {
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

	armMutexKV.Lock(name)
	defer armMutexKV.Unlock(name)

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

		networkSecurityGroupName, err := parseNetworkSecurityGroupName(nsgId)
		if err != nil {
			return err
		}

		armMutexKV.Lock(networkSecurityGroupName)
		defer armMutexKV.Unlock(networkSecurityGroupName)
	}

	if v, ok := d.GetOk("route_table_id"); ok {
		rtId := v.(string)
		properties.RouteTable = &network.RouteTable{
			ID: &rtId,
		}

		routeTableName, err := parseRouteTableName(rtId)
		if err != nil {
			return err
		}

		armMutexKV.Lock(routeTableName)
		defer armMutexKV.Unlock(routeTableName)
	}

	subnet := network.Subnet{
		Name: &name,
		SubnetPropertiesFormat: &properties,
	}

	_, error := subnetClient.CreateOrUpdate(resGroup, vnetName, name, subnet, make(chan struct{}))
	err := <-error
	if err != nil {
		return err
	}

	read, err := subnetClient.Get(resGroup, vnetName, name, "")
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Subnet %s/%s (resource group %s) ID", vnetName, name, resGroup)
	}

	d.SetId(*read.ID)

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

	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure Subnet %s: %s", name, err)
	}

	d.Set("name", name)
	d.Set("resource_group_name", resGroup)
	d.Set("virtual_network_name", vnetName)
	d.Set("address_prefix", resp.SubnetPropertiesFormat.AddressPrefix)

	if resp.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
		d.Set("network_security_group_id", resp.SubnetPropertiesFormat.NetworkSecurityGroup.ID)
	}

	if resp.SubnetPropertiesFormat.RouteTable != nil {
		d.Set("route_table_id", resp.SubnetPropertiesFormat.RouteTable.ID)
	}

	if resp.SubnetPropertiesFormat.IPConfigurations != nil {
		ips := make([]string, 0, len(*resp.SubnetPropertiesFormat.IPConfigurations))
		for _, ip := range *resp.SubnetPropertiesFormat.IPConfigurations {
			ips = append(ips, *ip.ID)
		}

		if err := d.Set("ip_configurations", ips); err != nil {
			return err
		}
	} else {
		d.Set("ip_configurations", []string{})
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

	if v, ok := d.GetOk("network_security_group_id"); ok {
		networkSecurityGroupId := v.(string)
		networkSecurityGroupName, err := parseNetworkSecurityGroupName(networkSecurityGroupId)
		if err != nil {
			return err
		}

		armMutexKV.Lock(networkSecurityGroupName)
		defer armMutexKV.Unlock(networkSecurityGroupName)
	}

	if v, ok := d.GetOk("route_table_id"); ok {
		rtId := v.(string)
		routeTableName, err := parseRouteTableName(rtId)
		if err != nil {
			return err
		}

		armMutexKV.Lock(routeTableName)
		defer armMutexKV.Unlock(routeTableName)
	}

	armMutexKV.Lock(vnetName)
	defer armMutexKV.Unlock(vnetName)

	armMutexKV.Lock(name)
	defer armMutexKV.Unlock(name)

	_, error := subnetClient.Delete(resGroup, vnetName, name, make(chan struct{}))
	err = <-error

	return err
}
