package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsInstances() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsInstancesRead,

		Schema: map[string]*schema.Schema{
			"filter":        dataSourceFiltersSchema(),
			"instance_tags": tagsSchemaComputed(),

			"ids": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"private_ips": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"public_ips": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceAwsInstancesRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	filters, filtersOk := d.GetOk("filter")
	tags, tagsOk := d.GetOk("instance_tags")

	if !filtersOk && !tagsOk {
		return fmt.Errorf("One of filters or instance_tags must be assigned")
	}

	params := &ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   aws.String("instance-state-name"),
				Values: []*string{aws.String("running")},
			},
		},
	}
	if filtersOk {
		params.Filters = append(params.Filters,
			buildAwsDataSourceFilters(filters.(*schema.Set))...)
	}
	if tagsOk {
		params.Filters = append(params.Filters, buildEC2TagFilterList(
			tagsFromMap(tags.(map[string]interface{})),
		)...)
	}

	log.Printf("[DEBUG] Reading EC2 instances: %s", params)

	var instanceIds, privateIps, publicIps []string
	err := conn.DescribeInstancesPages(params, func(resp *ec2.DescribeInstancesOutput, isLast bool) bool {
		for _, res := range resp.Reservations {
			for _, instance := range res.Instances {
				instanceIds = append(instanceIds, *instance.InstanceId)
				if instance.PrivateIpAddress != nil {
					privateIps = append(privateIps, *instance.PrivateIpAddress)
				}
				if instance.PublicIpAddress != nil {
					publicIps = append(publicIps, *instance.PublicIpAddress)
				}
			}
		}
		return !isLast
	})
	if err != nil {
		return err
	}

	if len(instanceIds) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}

	log.Printf("[DEBUG] Found %d instances via given filter", len(instanceIds))

	d.SetId(resource.UniqueId())
	err = d.Set("ids", instanceIds)
	if err != nil {
		return err
	}

	err = d.Set("private_ips", privateIps)
	if err != nil {
		return err
	}

	err = d.Set("public_ips", publicIps)
	if err != nil {
		return err
	}

	return nil
}
