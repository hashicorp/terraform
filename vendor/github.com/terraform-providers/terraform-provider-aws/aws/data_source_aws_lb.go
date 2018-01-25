package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsLb() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsLbRead,
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

			"load_balancer_type": {
				Type:     schema.TypeString,
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

			"subnet_mapping": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"subnet_id": {
							Type:     schema.TypeString,
							Required: true,
						},
						"allocation_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
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

func dataSourceAwsLbRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn
	lbArn := d.Get("arn").(string)
	lbName := d.Get("name").(string)

	describeLbOpts := &elbv2.DescribeLoadBalancersInput{}
	switch {
	case lbArn != "":
		describeLbOpts.LoadBalancerArns = []*string{aws.String(lbArn)}
	case lbName != "":
		describeLbOpts.Names = []*string{aws.String(lbName)}
	}

	log.Printf("[DEBUG] Reading Load Balancer: %s", describeLbOpts)
	describeResp, err := elbconn.DescribeLoadBalancers(describeLbOpts)
	if err != nil {
		return errwrap.Wrapf("Error retrieving LB: {{err}}", err)
	}
	if len(describeResp.LoadBalancers) != 1 {
		return fmt.Errorf("Search returned %d results, please revise so only one is returned", len(describeResp.LoadBalancers))
	}
	d.SetId(*describeResp.LoadBalancers[0].LoadBalancerArn)

	return flattenAwsLbResource(d, meta, describeResp.LoadBalancers[0])
}
