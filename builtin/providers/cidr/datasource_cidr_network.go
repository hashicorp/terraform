package cidr

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"

	goc "github.com/apparentlymart/go-cidr/cidr"
	"github.com/hashicorp/terraform/helper/schema"
)

type subnetMask struct {
	Name string
	Mask int
}

func dataSourceNetwork() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceNetworkRead,

		Schema: map[string]*schema.Schema{
			"cidr_block": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The CIDR Block for the entire network (aka supernet)",
				Required:    true,
			},
			"subnet": &schema.Schema{
				Type:        schema.TypeList,
				Description: "The desired subnet masks to create",
				Required:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"mask": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
			"subnet_cidrs": &schema.Schema{
				Type:        schema.TypeMap,
				Description: "Map keyed by subnet name returning the value of the network in CIDR notation",
				Computed:    true,
			},
		},
	}
}

func dataSourceNetworkRead(d *schema.ResourceData, meta interface{}) error {
	var masterCIDRList []*net.IPNet
	rawSubnetMasks := d.Get("subnet").([]interface{})
	subnetMasks := expandSubnetMasks(rawSubnetMasks)
	cBlock := d.Get("cidr_block").(string)
	_, startNet, err := net.ParseCIDR(cBlock)
	if err != nil {
		return fmt.Errorf("Error parsing CIDR %v for cidr_network\n", err)
	}
	d.SetId(hashID(cBlock, subnetMasks))

	currentSubnet, _ := goc.PreviousSubnet(startNet, subnetMasks[0].Mask)
	subnetCIDRs, subnets, err := calculateSubnets(currentSubnet, subnetMasks)
	if err != nil {
		return fmt.Errorf("Error [ %v ] calculating subnets for cidr_network\n", err)
	}
	masterCIDRList = subnets
	d.Set("subnet_cidrs", subnetCIDRs)
	networkErr := goc.VerifyNoOverlap(masterCIDRList, startNet)
	if networkErr != nil {
		return networkErr
	}
	return nil
}

func calculateSubnets(currentSubnet *net.IPNet,
	subnetMasks []*subnetMask) (map[string]string, []*net.IPNet, error) {

	subnetCIDRs := make(map[string]string)
	subnets := make([]*net.IPNet, len(subnetMasks))
	for i, s := range subnetMasks {
		tmpSubnet, rollover := goc.NextSubnet(currentSubnet, s.Mask)
		if rollover {
			return nil, nil, fmt.Errorf("Next from %s exceeded maximum value\n", currentSubnet.String())
		}
		currentSubnet = tmpSubnet
		subnets[i] = currentSubnet
		subnetCIDRs[s.Name] = currentSubnet.String()
	}
	return subnetCIDRs, subnets, nil
}

func expandSubnetMasks(rawSubnetMasks []interface{}) []*subnetMask {
	subnetMasks := make([]*subnetMask, len(rawSubnetMasks))
	for i, sRaw := range rawSubnetMasks {
		data := sRaw.(map[string]interface{})
		subnetMask := &subnetMask{
			Mask: data["mask"].(int),
			Name: data["name"].(string),
		}
		subnetMasks[i] = subnetMask
	}
	return subnetMasks
}

func hashID(cBlock string, subnetMasks []*subnetMask) string {
	h := sha256.New()
	h.Write([]byte(cBlock))
	for _, s := range subnetMasks {
		h.Write([]byte(s.Name))
		h.Write([]byte(strconv.Itoa(s.Mask)))
	}
	return hex.EncodeToString(h.Sum(nil))
}
