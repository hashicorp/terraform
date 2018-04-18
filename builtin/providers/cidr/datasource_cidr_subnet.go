package cidr

import (
	"fmt"
	"net"
	"strconv"

	goc "github.com/apparentlymart/go-cidr/cidr"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceSubnet() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceSubnetRead,

		Schema: map[string]*schema.Schema{
			"cidr_block": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The CIDR block for the entire network (aka supernet)",
				Required:    true,
			},
			"start_after": &schema.Schema{
				Type:        schema.TypeString,
				Description: "The subnet in CIDR notation to offset subnet creation by",
				Optional:    true,
			},
			"subnet_count": &schema.Schema{
				Type:        schema.TypeInt,
				Description: "The number of subnets to create",
				Default:     1,
				Optional:    true,
			},
			"subnet_mask": &schema.Schema{
				Type:        schema.TypeInt,
				Description: "The desired subnet mask to use for creation",
				Required:    true,
			},
			"max_subnet": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"subnet_cidrs": &schema.Schema{
				Type:        schema.TypeList,
				Description: "The set of subnets in CIDR notation",
				Optional:    true,
				Computed:    true,
				Elem:        &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceSubnetRead(d *schema.ResourceData, meta interface{}) error {
	d.SetId(subnetId(d))
	subnetCount := d.Get("subnet_count").(int)
	subnets := make([]*net.IPNet, subnetCount)
	subnetCIDRs := make([]string, subnetCount)
	mask := d.Get("subnet_mask").(int)

	_, startNet, perr := net.ParseCIDR(d.Get("cidr_block").(string))
	if perr != nil {
		return fmt.Errorf("cidr_block parse error %v\n", perr)
	}

	var currentSubnet *net.IPNet
	startAfter, sap := d.GetOk("start_after")
	if sap {
		_, offsetSubnet, perr := net.ParseCIDR(startAfter.(string))
		if perr != nil {
			return fmt.Errorf("start_after %v resulted in parse error %v\n", startAfter, perr)
		}
		currentSubnet = offsetSubnet
	} else {
		currentSubnet, _ = goc.PreviousSubnet(startNet, mask)
	}
	for i := 0; i < subnetCount; i++ {
		tmpSubnet, rollover := goc.NextSubnet(currentSubnet, mask)
		if rollover {
			return fmt.Errorf("Next from %s exceeded maximum value\n", currentSubnet.String())
		}
		currentSubnet = tmpSubnet
		subnets[i] = currentSubnet
		subnetCIDRs[i] = currentSubnet.String()
	}
	nerr := goc.VerifyNoOverlap(subnets, startNet)
	if nerr != nil {
		return fmt.Errorf("Network is invalid: [ %v ]", nerr)
	}

	d.Set("subnet_cidrs", subnetCIDRs)
	d.Set("max_subnet", subnetCIDRs[subnetCount-1])
	return nil
}

func subnetId(d *schema.ResourceData) string {
	id := d.Get("cidr_block").(string)
	startAfter, sap := d.GetOk("start_after")
	if sap {
		id = id + startAfter.(string)
	}
	subnetCount, scp := d.GetOk("subnet_count")
	if scp {
		id = id + strconv.Itoa(subnetCount.(int))
	}
	mask := d.Get("subnet_mask").(int)
	id = id + strconv.Itoa(mask)
	return strconv.Itoa(hashcode.String(id))
}
