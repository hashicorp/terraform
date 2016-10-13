package azurerm

import (
	"fmt"
	"net/http"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmRoute() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmRouteCreate,
		Read:   resourceArmRouteRead,
		Update: resourceArmRouteCreate,
		Delete: resourceArmRouteDelete,

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

			"route_table_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"address_prefix": {
				Type:     schema.TypeString,
				Required: true,
			},

			"next_hop_type": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateRouteTableNextHopType,
			},

			"next_hop_in_ip_address": {
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

	_, err := routesClient.CreateOrUpdate(resGroup, rtName, name, route, make(chan struct{}))
	if err != nil {
		return err
	}

	read, err := routesClient.Get(resGroup, rtName, name)
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Route %s/%s (resource group %s) ID", rtName, name, resGroup)
	}
	d.SetId(*read.ID)

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
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
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

	_, err = routesClient.Delete(resGroup, rtName, routeName, make(chan struct{}))

	return err
}
