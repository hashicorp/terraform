package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsElb() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsElbRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"access_logs": {
				Type:     schema.TypeList,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"interval": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"bucket": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"bucket_prefix": {
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

			"availability_zones": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},

			"connection_draining": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"connection_draining_timeout": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"cross_zone_load_balancing": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"dns_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"health_check": {
				Type:     schema.TypeList,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"healthy_threshold": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"unhealthy_threshold": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"target": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"interval": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"timeout": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},

			"idle_timeout": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"instances": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},

			"internal": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"listener": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"instance_port": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"instance_protocol": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"lb_port": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"lb_protocol": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"ssl_certificate_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
				Set: resourceAwsElbListenerHash,
			},

			"security_groups": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},

			"source_security_group": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"source_security_group_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"subnets": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},

			"tags": tagsSchemaComputed(),

			"zone_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsElbRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbconn
	lbName := d.Get("name").(string)

	input := &elb.DescribeLoadBalancersInput{
		LoadBalancerNames: []*string{aws.String(lbName)},
	}

	log.Printf("[DEBUG] Reading ELB: %s", input)
	resp, err := elbconn.DescribeLoadBalancers(input)
	if err != nil {
		return fmt.Errorf("Error retrieving LB: %s", err)
	}
	if len(resp.LoadBalancerDescriptions) != 1 {
		return fmt.Errorf("Search returned %d results, please revise so only one is returned", len(resp.LoadBalancerDescriptions))
	}
	d.SetId(*resp.LoadBalancerDescriptions[0].LoadBalancerName)

	return flattenAwsELbResource(d, meta.(*AWSClient).ec2conn, elbconn, resp.LoadBalancerDescriptions[0])
}
