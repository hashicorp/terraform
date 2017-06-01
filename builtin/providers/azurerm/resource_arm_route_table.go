package azurerm

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/Azure/azure-sdk-for-go/arm/network"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceArmRouteTable() *schema.Resource {
	return &schema.Resource{
		Create: resourceArmRouteTableCreate,
		Read:   resourceArmRouteTableRead,
		Update: resourceArmRouteTableCreate,
		Delete: resourceArmRouteTableDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"location": locationSchema(),

			"resource_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"route": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
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
				},
				Set: resourceArmRouteTableRouteHash,
			},

			"subnets": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceArmRouteTableCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ArmClient)
	routeTablesClient := client.routeTablesClient

	log.Printf("[INFO] preparing arguments for Azure ARM Route Table creation.")

	name := d.Get("name").(string)
	location := d.Get("location").(string)
	resGroup := d.Get("resource_group_name").(string)
	tags := d.Get("tags").(map[string]interface{})

	routeSet := network.RouteTable{
		Name:     &name,
		Location: &location,
		Tags:     expandTags(tags),
	}

	if _, ok := d.GetOk("route"); ok {
		routes, routeErr := expandAzureRmRouteTableRoutes(d)
		if routeErr != nil {
			return fmt.Errorf("Error Building list of Route Table Routes: %s", routeErr)
		}

		if len(routes) > 0 {
			routeSet.RouteTablePropertiesFormat = &network.RouteTablePropertiesFormat{
				Routes: &routes,
			}
		}
	}

	_, error := routeTablesClient.CreateOrUpdate(resGroup, name, routeSet, make(chan struct{}))
	err := <-error
	if err != nil {
		return err
	}

	read, err := routeTablesClient.Get(resGroup, name, "")
	if err != nil {
		return err
	}
	if read.ID == nil {
		return fmt.Errorf("Cannot read Route Table %s (resource group %s) ID", name, resGroup)
	}

	d.SetId(*read.ID)

	return resourceArmRouteTableRead(d, meta)
}

func resourceArmRouteTableRead(d *schema.ResourceData, meta interface{}) error {
	routeTablesClient := meta.(*ArmClient).routeTablesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["routeTables"]

	resp, err := routeTablesClient.Get(resGroup, name, "")
	if err != nil {
		if resp.StatusCode == http.StatusNotFound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error making Read request on Azure Route Table %s: %s", name, err)
	}

	d.Set("name", name)
	d.Set("resource_group_name", resGroup)
	d.Set("location", resp.Location)

	if resp.RouteTablePropertiesFormat.Routes != nil {
		d.Set("route", schema.NewSet(resourceArmRouteTableRouteHash, flattenAzureRmRouteTableRoutes(resp.RouteTablePropertiesFormat.Routes)))
	}

	subnets := []string{}
	if resp.RouteTablePropertiesFormat.Subnets != nil {
		for _, subnet := range *resp.RouteTablePropertiesFormat.Subnets {
			id := subnet.ID
			subnets = append(subnets, *id)
		}
	}
	d.Set("subnets", subnets)

	flattenAndSetTags(d, resp.Tags)

	return nil
}

func resourceArmRouteTableDelete(d *schema.ResourceData, meta interface{}) error {
	routeTablesClient := meta.(*ArmClient).routeTablesClient

	id, err := parseAzureResourceID(d.Id())
	if err != nil {
		return err
	}
	resGroup := id.ResourceGroup
	name := id.Path["routeTables"]

	_, error := routeTablesClient.Delete(resGroup, name, make(chan struct{}))
	err = <-error

	return err
}

func expandAzureRmRouteTableRoutes(d *schema.ResourceData) ([]network.Route, error) {
	configs := d.Get("route").(*schema.Set).List()
	routes := make([]network.Route, 0, len(configs))

	for _, configRaw := range configs {
		data := configRaw.(map[string]interface{})

		address_prefix := data["address_prefix"].(string)
		next_hop_type := data["next_hop_type"].(string)

		properties := network.RoutePropertiesFormat{
			AddressPrefix: &address_prefix,
			NextHopType:   network.RouteNextHopType(next_hop_type),
		}

		if v := data["next_hop_in_ip_address"].(string); v != "" {
			properties.NextHopIPAddress = &v
		}

		name := data["name"].(string)
		route := network.Route{
			Name: &name,
			RoutePropertiesFormat: &properties,
		}

		routes = append(routes, route)
	}

	return routes, nil
}

func flattenAzureRmRouteTableRoutes(routes *[]network.Route) []interface{} {
	results := make([]interface{}, 0, len(*routes))

	for _, route := range *routes {
		r := make(map[string]interface{})
		r["name"] = *route.Name
		r["address_prefix"] = *route.RoutePropertiesFormat.AddressPrefix
		r["next_hop_type"] = string(route.RoutePropertiesFormat.NextHopType)
		if route.RoutePropertiesFormat.NextHopIPAddress != nil {
			r["next_hop_in_ip_address"] = *route.RoutePropertiesFormat.NextHopIPAddress
		}
		results = append(results, r)
	}

	return results
}

func resourceArmRouteTableRouteHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["address_prefix"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", strings.ToLower(m["next_hop_type"].(string))))

	return hashcode.String(buf.String())
}

func validateRouteTableNextHopType(v interface{}, k string) (ws []string, errors []error) {
	value := strings.ToLower(v.(string))
	hopTypes := map[string]bool{
		"virtualnetworkgateway": true,
		"vnetlocal":             true,
		"internet":              true,
		"virtualappliance":      true,
		"none":                  true,
	}

	if !hopTypes[value] {
		errors = append(errors, fmt.Errorf("Route Table NextHopType Protocol can only be VirtualNetworkGateway, VnetLocal, Internet or VirtualAppliance"))
	}
	return
}
