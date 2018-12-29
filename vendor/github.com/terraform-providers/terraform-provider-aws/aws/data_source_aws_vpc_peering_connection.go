package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsVpcPeeringConnection() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsVpcPeeringConnectionRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"status": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"owner_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"cidr_block": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"peer_vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"peer_owner_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"peer_cidr_block": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"peer_region": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"accepter": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     schema.TypeBool,
			},
			"requester": {
				Type:     schema.TypeMap,
				Computed: true,
				Elem:     schema.TypeBool,
			},
			"filter": ec2CustomFiltersSchema(),
			"tags":   tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsVpcPeeringConnectionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Reading VPC Peering Connections.")

	req := &ec2.DescribeVpcPeeringConnectionsInput{}

	if id, ok := d.GetOk("id"); ok {
		req.VpcPeeringConnectionIds = aws.StringSlice([]string{id.(string)})
	}

	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"status-code":                   d.Get("status").(string),
			"requester-vpc-info.vpc-id":     d.Get("vpc_id").(string),
			"requester-vpc-info.owner-id":   d.Get("owner_id").(string),
			"requester-vpc-info.cidr-block": d.Get("cidr_block").(string),
			"accepter-vpc-info.vpc-id":      d.Get("peer_vpc_id").(string),
			"accepter-vpc-info.owner-id":    d.Get("peer_owner_id").(string),
			"accepter-vpc-info.cidr-block":  d.Get("peer_cidr_block").(string),
		},
	)
	req.Filters = append(req.Filters, buildEC2TagFilterList(
		tagsFromMap(d.Get("tags").(map[string]interface{})),
	)...)
	req.Filters = append(req.Filters, buildEC2CustomFilterList(
		d.Get("filter").(*schema.Set),
	)...)
	if len(req.Filters) == 0 {
		// Don't send an empty filters list; the EC2 API won't accept it.
		req.Filters = nil
	}

	log.Printf("[DEBUG] Reading VPC Peering Connection: %s", req)
	resp, err := conn.DescribeVpcPeeringConnections(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.VpcPeeringConnections) == 0 {
		return fmt.Errorf("no matching VPC peering connection found")
	}
	if len(resp.VpcPeeringConnections) > 1 {
		return fmt.Errorf("multiple VPC peering connections matched; use additional constraints to reduce matches to a single VPC peering connection")
	}

	pcx := resp.VpcPeeringConnections[0]

	d.SetId(aws.StringValue(pcx.VpcPeeringConnectionId))
	d.Set("status", pcx.Status.Code)
	d.Set("vpc_id", pcx.RequesterVpcInfo.VpcId)
	d.Set("owner_id", pcx.RequesterVpcInfo.OwnerId)
	d.Set("cidr_block", pcx.RequesterVpcInfo.CidrBlock)
	d.Set("region", pcx.RequesterVpcInfo.Region)
	d.Set("peer_vpc_id", pcx.AccepterVpcInfo.VpcId)
	d.Set("peer_owner_id", pcx.AccepterVpcInfo.OwnerId)
	d.Set("peer_cidr_block", pcx.AccepterVpcInfo.CidrBlock)
	d.Set("peer_region", pcx.AccepterVpcInfo.Region)
	d.Set("tags", tagsToMap(pcx.Tags))

	if pcx.AccepterVpcInfo.PeeringOptions != nil {
		if err := d.Set("accepter", flattenPeeringOptions(pcx.AccepterVpcInfo.PeeringOptions)[0]); err != nil {
			return err
		}
	}

	if pcx.RequesterVpcInfo.PeeringOptions != nil {
		if err := d.Set("requester", flattenPeeringOptions(pcx.RequesterVpcInfo.PeeringOptions)[0]); err != nil {
			return err
		}
	}

	return nil
}
