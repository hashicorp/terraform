package ibmcloud

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
)

func dataSourceIBMCloudInfraVlan() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceIBMCloudInfraVlanRead,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"number": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			"router_hostname": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"subnets": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceIBMCloudInfraVlanRead(d *schema.ResourceData, meta interface{}) error {
	sess := meta.(ClientSession).SoftLayerSession()
	service := services.GetAccountService(sess)

	number, numberOk := d.GetOk("number")
	var vlan *datatypes.Network_Vlan
	var err error
	routerHostname, rHostNameOk := d.GetOk("router_hostname")

	if numberOk && rHostNameOk {
		// Got vlan number and router, get vlan, and compute name
		vlan, err = getVlan(number.(int), routerHostname.(string), meta)
		if err != nil {
			return err
		}

		d.SetId(fmt.Sprintf("%d", *vlan.Id))
		if vlan.Name != nil {
			d.Set("name", vlan.Name)
		}
	} else if name, ok := d.GetOk("name"); ok {
		// Got name, get vlan, and compute router hostname and vlan number
		networkVlans, err := service.
			Mask("id,vlanNumber,name,primaryRouter[hostname],primarySubnets[networkIdentifier,cidr]").
			Filter(filter.Path("networkVlans.name").Eq(name.(string)).Build()).
			GetNetworkVlans()
		if err != nil {
			return fmt.Errorf("Error obtaining VLAN id: %s", err)
		}
		if len(networkVlans) == 0 {
			return fmt.Errorf("No VLAN was found with the name '%s'", name.(string))
		}

		vlan = &networkVlans[0]
		d.SetId(fmt.Sprintf("%d", *vlan.Id))
		d.Set("number", vlan.VlanNumber)

		if vlan.PrimaryRouter != nil && vlan.PrimaryRouter.Hostname != nil {
			d.Set("router_hostname", vlan.PrimaryRouter.Hostname)
		}
	} else {
		return errors.New("Missing required properties. Need a VLAN name, or the VLAN's number and router hostname")
	}

	// Get subnets in cidr format for display
	if len(vlan.PrimarySubnets) > 0 {
		subnets := make([]string, len(vlan.PrimarySubnets))
		for i, subnet := range vlan.PrimarySubnets {
			subnets[i] = fmt.Sprintf("%s/%d", *subnet.NetworkIdentifier, *subnet.Cidr)
		}

		d.Set("subnets", subnets)
	}

	return nil
}

func getVlan(vlanNumber int, primaryRouterHostname string, meta interface{}) (*datatypes.Network_Vlan, error) {
	service := services.GetAccountService(meta.(ClientSession).SoftLayerSession())

	networkVlans, err := service.
		Mask("id,name,primarySubnets[networkIdentifier,cidr]").
		Filter(
			filter.Build(
				filter.Path("networkVlans.primaryRouter.hostname").Eq(primaryRouterHostname),
				filter.Path("networkVlans.vlanNumber").Eq(vlanNumber),
			),
		).
		GetNetworkVlans()

	if err != nil {
		return &datatypes.Network_Vlan{}, fmt.Errorf("Error looking up Vlan: %s", err)
	}

	if len(networkVlans) < 1 {
		return &datatypes.Network_Vlan{}, fmt.Errorf(
			"Unable to locate a vlan matching the provided router hostname and vlan number: %s/%d",
			primaryRouterHostname,
			vlanNumber)
	}

	return &networkVlans[0], nil
}
