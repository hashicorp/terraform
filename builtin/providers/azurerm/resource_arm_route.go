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

func resourceArmRoute() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmRouteCreate,
		Read:   resourceArmRouteRead,
		Update: resourceArmRouteCreate,
		Delete: resourceArmRouteDelete,

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

			"route_table_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"address_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"next_hop_type": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRouteTableNextHopType,
			},

			"next_hop_in_ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceArmRouteCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	routesClient := client.routesClient

	name := d.Get("name").(string)
	rtName := d.Get("route_table_name").(string)
	resGroup := d.Get("resource_group_name").(string)

	addressPrefix := d.Get("address_prefix").(string)
	nextHopType := d.Get("next_hop_type").(string)

	armMutexKV.Lock(rtName)
	defer armMutexKV.Unlock(rtName)

	properties := network.RoutePropertiesFormat{
		AddressPrefix: &addressPrefix,
		NextHopType:   network.RouteNextHopType(nextHopType),
	}

	if v, ok := d.GetOk("next_hop_in_ip_address"); ok {
		nextHopInIpAddress := v.(string)
		properties.NextHopIPAddress = &nextHopInIpAddress
	}

	route := network.Route{
		Name:       &name,
		Properties: &properties,
	}

	resp, err := routesClient.CreateOrUpdate(resGroup, rtName, name, route)
	if err != nil {
		return err
	}
	d.SetId(*resp.ID)

	log.Printf("[DEBUG] Waiting for Route (%s) to become available", name)
	stateConf := &resource.StateChangeConf{
		Pending: []string{"Accepted", "Updating"},
		Target:  []string{"Succeeded"},
		Refresh: routeStateRefreshFunc(client, resGroup, rtName, name),
		Timeout: 10 * time.Minute,
	}
	if _, err := stateConf.WaitForState(); err != nil {
		return fmt.Errorf("Error waiting for Route (%s) to become available: %s", name, err)
	}

	return resourceArmRouteRead(d, meta)
}

func resourceArmRouteRead(d *schema.ResourceData, meta interface{}) error {
	routesClient := meta.(*ArmClient).routesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	rtName := id.Path["routeTables"]
	routeName := id.Path["routes"]

	resp, err := routesClient.Get(resGroup, rtName, routeName)
	if resp.StatusCode == http.StatusNotFound {
		d.SetId("")
		return nil
	}
	if err != nil {
		return fmt.Errorf("Error making Read request on Azure Route %s: %s", routeName, err)
	}

	return nil
}

func resourceArmRouteDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	routesClient := client.routesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	rtName := id.Path["routeTables"]
	routeName := id.Path["routes"]

	armMutexKV.Lock(rtName)
	defer armMutexKV.Unlock(rtName)

	_, err = routesClient.Delete(resGroup, rtName, routeName)

	return err
}

func routeStateRefreshFunc(client *ArmClient, resourceGroupName string, routeTableName string, routeName string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		res, err := client.routesClient.Get(resourceGroupName, routeTableName, routeName)
		if err != nil {
			return nil, "", fmt.Errorf("Error issuing read request in routeStateRefreshFunc to Azure ARM for route '%s' (RG: '%s') (NSG: '%s'): %s", routeName, resourceGroupName, routeTableName, err)
		}

		return res, *res.Properties.ProvisioningState, nil
	}
}
