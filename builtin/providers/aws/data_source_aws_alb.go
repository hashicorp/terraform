package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsAlb() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsAlbRead,
		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"arn_suffix": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"internal": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"security_groups": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},

			"subnets": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},

			"access_logs": {
				Type:     schema.TypeList,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"prefix": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},

			"enable_deletion_protection": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"idle_timeout": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"zone_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"dns_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsAlbRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn
	albArn := d.Get("arn").(string)
	albName := d.Get("name").(string)

	describeAlbOpts := &elbv2.DescribeLoadBalancersInput{}
	switch {
	case albArn != "":
		describeAlbOpts.LoadBalancerArns = []*string{aws.String(albArn)}
	case albName != "":
		describeAlbOpts.Names = []*string{aws.String(albName)}
	}

	describeResp, err := elbconn.DescribeLoadBalancers(describeAlbOpts)
	if err != nil {
		return errwrap.Wrapf("Error retrieving ALB: {{err}}", err)
	}
	if len(describeResp.LoadBalancers) != 1 {
		return fmt.Errorf("Search returned %d results, please revise so only one is returned", len(describeResp.LoadBalancers))
	}
	d.SetId(*describeResp.LoadBalancers[0].LoadBalancerArn)

	return flattenAwsAlbResource(d, meta, describeResp.LoadBalancers[0])
}
