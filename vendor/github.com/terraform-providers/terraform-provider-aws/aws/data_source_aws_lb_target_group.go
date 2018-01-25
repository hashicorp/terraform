package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsLbTargetGroup() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsLbTargetGroupRead,
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

			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"protocol": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"deregistration_delay": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"stickiness": {
				Type:     schema.TypeList,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"cookie_duration": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},

			"health_check": {
				Type:     schema.TypeList,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"interval": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"path": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"port": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"protocol": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"timeout": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"healthy_threshold": {
							Type:     schema.TypeInt,
							Computed: true,
						},

						"matcher": {
							Type:     schema.TypeString,
							Computed: true,
						},

						"unhealthy_threshold": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},

			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsLbTargetGroupRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn
	tgArn := d.Get("arn").(string)
	tgName := d.Get("name").(string)

	describeTgOpts := &elbv2.DescribeTargetGroupsInput{}
	switch {
	case tgArn != "":
		describeTgOpts.TargetGroupArns = []*string{aws.String(tgArn)}
	case tgName != "":
		describeTgOpts.Names = []*string{aws.String(tgName)}
	}

	log.Printf("[DEBUG] Reading Load Balancer Target Group: %s", describeTgOpts)
	describeResp, err := elbconn.DescribeTargetGroups(describeTgOpts)
	if err != nil {
		return errwrap.Wrapf("Error retrieving LB Target Group: {{err}}", err)
	}
	if len(describeResp.TargetGroups) != 1 {
		return fmt.Errorf("Search returned %d results, please revise so only one is returned", len(describeResp.TargetGroups))
	}

	targetGroup := describeResp.TargetGroups[0]

	d.SetId(*targetGroup.TargetGroupArn)
	return flattenAwsLbTargetGroupResource(d, meta, targetGroup)
}
