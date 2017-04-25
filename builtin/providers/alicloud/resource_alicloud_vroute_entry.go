package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
)

func resourceAliyunRouteEntry() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunRouteEntryCreate,
		Read:   resourceAliyunRouteEntryRead,
		Delete: resourceAliyunRouteEntryDelete,

		Schema: map[string]*schema.Schema{
			"router_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"route_table_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"destination_cidrblock": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"nexthop_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateRouteEntryNextHopType,
			},
			"nexthop_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAliyunRouteEntryCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	rtId := d.Get("route_table_id").(string)
	rId := d.Get("router_id").(string)
	cidr := d.Get("destination_cidrblock").(string)
	nt := d.Get("nexthop_type").(string)
	ni := d.Get("nexthop_id").(string)

	args, err := buildAliyunRouteEntryArgs(d, meta)
	if err != nil {
		return err
	}
	err = conn.CreateRouteEntry(args)

	if err != nil {
		return err
	}
	// route_table_id:router_id:destination_cidrblock:nexthop_type:nexthop_id
	d.SetId(rtId + ":" + rId + ":" + cidr + ":" + nt + ":" + ni)
	d.Set("router_id", rId)

	if err := conn.WaitForAllRouteEntriesAvailable(rId, rtId, defaultTimeout); err != nil {
		return fmt.Errorf("WaitFor route entry got error: %#v", err)
	}
	return resourceAliyunRouteEntryRead(d, meta)
}

func resourceAliyunRouteEntryRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	parts := strings.Split(d.Id(), ":")
	rtId := parts[0]
	//rId := parts[1]
	cidr := parts[2]
	nexthop_type := parts[3]
	nexthop_id := parts[4]

	en, err := client.QueryRouteEntry(rtId, cidr, nexthop_type, nexthop_id)

	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error route entry: %#v", err)
	}

	d.Set("route_table_id", en.RouteTableId)
	d.Set("destination_cidrblock", en.DestinationCidrBlock)
	d.Set("nexthop_type", en.NextHopType)
	d.Set("nexthop_id", en.InstanceId)
	return nil
}

func resourceAliyunRouteEntryDelete(d *schema.ResourceData, meta interface{}) error {
	con := meta.(*AliyunClient).ecsconn
	args, err := buildAliyunRouteEntryDeleteArgs(d, meta)

	if err != nil {
		return err
	}
	return con.DeleteRouteEntry(args)
}

func buildAliyunRouteEntryArgs(d *schema.ResourceData, meta interface{}) (*ecs.CreateRouteEntryArgs, error) {

	args := &ecs.CreateRouteEntryArgs{
		RouteTableId:         d.Get("route_table_id").(string),
		DestinationCidrBlock: d.Get("destination_cidrblock").(string),
	}

	if v := d.Get("nexthop_type").(string); v != "" {
		args.NextHopType = ecs.NextHopType(v)
	}

	if v := d.Get("nexthop_id").(string); v != "" {
		args.NextHopId = v
	}

	return args, nil
}

func buildAliyunRouteEntryDeleteArgs(d *schema.ResourceData, meta interface{}) (*ecs.DeleteRouteEntryArgs, error) {

	args := &ecs.DeleteRouteEntryArgs{
		RouteTableId:         d.Get("route_table_id").(string),
		DestinationCidrBlock: d.Get("destination_cidrblock").(string),
	}

	if v := d.Get("destination_cidrblock").(string); v != "" {
		args.DestinationCidrBlock = v
	}

	if v := d.Get("nexthop_id").(string); v != "" {
		args.NextHopId = v
	}

	return args, nil
}
