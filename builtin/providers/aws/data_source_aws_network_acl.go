package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsNetworkAcl() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsNetworkAclRead,

		Schema: map[string]*schema.Schema{
			"network_acl_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"vpc_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"default": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"subnet_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"filter": ec2CustomFiltersSchema(),
			"tags":   tagsSchemaComputed(),
			"ingress": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from_port": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"to_port": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"rule_no": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"action": {
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
						},
						"cidr_block": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ipv6_cidr_block": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"icmp_type": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"icmp_code": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
			"egress": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from_port": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"to_port": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"rule_no": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"action": {
							Type:     schema.TypeString,
							Required: true,
						},
						"protocol": {
							Type:     schema.TypeString,
							Required: true,
						},
						"cidr_block": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"ipv6_cidr_block": {
							Type:     schema.TypeString,
							Optional: true,
						},
						"icmp_type": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"icmp_code": {
							Type:     schema.TypeInt,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAwsNetworkAclRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeNetworkAclsInput{}

	if naclId, ok := d.GetOk("network_acl_id"); ok {
		req.NetworkAclIds = aws.StringSlice([]string{naclId.(string)})
	}

	isDefaultStr := ""
	if d.Get("default").(bool) {
		isDefaultStr = "true"
	}
	req.Filters = buildEC2AttributeFilterList(
		map[string]string{
			"default": isDefaultStr,
			"vpc-id":  d.Get("vpc_id").(string),
		},
	)
	if v, ok := d.GetOk("subnet_ids"); ok {
		var subnetIds []string
		ids := v.(*schema.Set).List()
		for _, id := range ids {
			subnetIds = append(subnetIds, id.(string))
		}

		req.Filters = append(req.Filters, &ec2.Filter{
			Name:   aws.String("association.subnet-id"),
			Values: aws.StringSlice(subnetIds),
		})
	}
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

	log.Printf("[DEBUG] Describe network ACLs %v\n", req)

	resp, err := conn.DescribeNetworkAcls(req)
	if err != nil {
		return err
	}
	if resp == nil || len(resp.NetworkAcls) == 0 {
		return fmt.Errorf("no matching network ACL found")
	}
	if len(resp.NetworkAcls) > 1 {
		return fmt.Errorf("multiple network ACLs matched; use additional constraints to reduce matches to a single network ACL")
	}

	networkAcl := resp.NetworkAcls[0]
	d.SetId(aws.StringValue(networkAcl.NetworkAclId))
	if err := awsNetworkAclAttributes(d, networkAcl); err != nil {
		return err
	}

	d.Set("network_acl_id", networkAcl.NetworkAclId)
	d.Set("default", networkAcl.IsDefault)

	return nil
}
