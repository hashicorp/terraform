package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func awsInstanceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"ami": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"instance_type": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"instance_state": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"availability_zone": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"tenancy": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"key_name": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"public_dns": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"public_ip": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"private_dns": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"private_ip": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"iam_instance_profile": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"subnet_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"network_interface_id": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"associate_public_ip_address": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"ebs_optimized": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"source_dest_check": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"monitoring": {
			Type:     schema.TypeBool,
			Computed: true,
		},
		"user_data": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"security_groups": {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"vpc_security_group_ids": {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"ephemeral_block_device": {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"device_name": {
						Type:     schema.TypeString,
						Required: true,
					},

					"virtual_name": {
						Type:     schema.TypeString,
						Optional: true,
					},

					"no_device": {
						Type:     schema.TypeBool,
						Optional: true,
					},
				},
			},
		},
		"ebs_block_device": {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"delete_on_termination": {
						Type:     schema.TypeBool,
						Computed: true,
					},

					"device_name": {
						Type:     schema.TypeString,
						Computed: true,
					},

					"encrypted": {
						Type:     schema.TypeBool,
						Computed: true,
					},

					"iops": {
						Type:     schema.TypeInt,
						Computed: true,
					},

					"snapshot_id": {
						Type:     schema.TypeString,
						Computed: true,
					},

					"volume_size": {
						Type:     schema.TypeInt,
						Computed: true,
					},

					"volume_type": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
			},
		},
		"root_block_device": {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"delete_on_termination": {
						Type:     schema.TypeBool,
						Computed: true,
					},

					"iops": {
						Type:     schema.TypeInt,
						Computed: true,
					},

					"volume_size": {
						Type:     schema.TypeInt,
						Computed: true,
					},

					"volume_type": {
						Type:     schema.TypeString,
						Computed: true,
					},
				},
			},
		},
	}
}

func dataSourceAwsInstance() *schema.Resource {
	s := map[string]*schema.Schema{
		"filter":        dataSourceFiltersSchema(),
		"tags":          dataSourceTagsSchema(),
		"instance_tags": tagsSchemaComputed(),
		"instance_id": {
			Type:     schema.TypeString,
			Optional: true,
			ForceNew: true,
		},
	}
	for k, v := range awsInstanceSchema() {
		s[k] = v
	}
	return &schema.Resource{
		Read: func(d *schema.ResourceData, meta interface{}) error {
			return dataSourceAwsInstanceRead(d, meta, false)
		},
		Schema: s,
	}
}

func dataSourceAwsInstances() *schema.Resource {
	return &schema.Resource{
		Read: func(d *schema.ResourceData, meta interface{}) error {
			return dataSourceAwsInstanceRead(d, meta, true)
		},
		Schema: map[string]*schema.Schema{
			"filter":        dataSourceFiltersSchema(),
			"tags":          dataSourceTagsSchema(),
			"instance_tags": tagsSchemaComputed(),
			"instance_ids": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"instances": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Resource{Schema: awsInstanceSchema()},
			},
		},
	}
}

func dataSourceAwsInstanceRead(d *schema.ResourceData, meta interface{}, multi bool) error {
	conn := meta.(*AWSClient).ec2conn

	filters, filtersOk := d.GetOk("filter")
	instanceID, instanceIDOk := d.GetOk("instance_id")
	if multi {
		instanceID, instanceIDOk = d.GetOk("instance_ids")
	}
	tags, tagsOk := d.GetOk("instance_tags")

	// Exit if none of the optional query parameters were provided
	if filtersOk == false && instanceIDOk == false && tagsOk == false {
		return fmt.Errorf("One of filters, instance_tags, or instance_id must be assigned")
	}

	// Build up search parameters
	params := &ec2.DescribeInstancesInput{}
	if filtersOk {
		params.Filters = buildAwsDataSourceFilters(filters.(*schema.Set))
	}
	if instanceIDOk && multi {
		for _, id := range instanceID.([]interface{}) {
			params.InstanceIds = append(params.InstanceIds, aws.String(id.(string)))
		}
	} else if instanceIDOk {
		params.InstanceIds = []*string{aws.String(instanceID.(string))}
	}
	if tagsOk {
		params.Filters = append(params.Filters, buildEC2TagFilterList(
			tagsFromMap(tags.(map[string]interface{})),
		)...)
	}

	// Perform the lookup
	resp, err := conn.DescribeInstances(params)
	if err != nil {
		return err
	}

	// Loop through reservations, and remove terminated instances, populate instance slice
	var filteredInstances []*ec2.Instance
	for _, res := range resp.Reservations {
		for _, instance := range res.Instances {
			if instance.State != nil && *instance.State.Name != "terminated" {
				filteredInstances = append(filteredInstances, instance)
			}
		}
	}

	// Handle case for aws_instance data source
	if !multi {
		if len(filteredInstances) < 1 {
			return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
		}
		if len(filteredInstances) > 1 {
			return fmt.Errorf("Your query returned more than one result. Please try a more specific search criteria.")
		}
		log.Printf("[DEBUG] aws_instance - Single Instance ID found: %s", *filteredInstances[0].InstanceId)
		if _, err := instanceDescriptionAttributes(d, filteredInstances[0], conn); err != nil {
			return err
		}
		return nil
	}

	// Handle case for aws_instances data source
	instances := make([]map[string]interface{}, 0, len(filteredInstances))
	instanceIds := make([]string, 0, len(filteredInstances))
	instanceResource := &schema.Resource{Schema: awsInstanceSchema()}
	instanceData := instanceResource.Data(nil)
	for _, i := range filteredInstances {
		// Use ResourceData, so we can reuse single instance attributes code
		if instanceData, err = instanceDescriptionAttributes(instanceData, i, conn); err != nil {
			return err
		}

		// Rebuild map from ResourceData, so it can be a part of aws_instances
		mapping := make(map[string]interface{})
		for k, _ := range instanceResource.Schema {
			if v, ok := instanceData.GetOk(k); ok {
				mapping[k] = v
			}
		}
		log.Printf("[DEBUG] aws_instances - Instance ID found: %s", mapping["id"])
		instances = append(instances, mapping)
		instanceIds = append(instanceIds, mapping["id"].(string))
	}

	log.Printf("[DEBUG] - ids: %s", instanceIds)
	log.Printf("[DEBUG] - instances: %s", instances)
	if err = d.Set("instances", instances); err != nil {
		return err
	}
	d.SetId(hashcode.Strings(instanceIds))
	return nil
}

// Populate instance attribute fields with the returned instance
func instanceDescriptionAttributes(d *schema.ResourceData, instance *ec2.Instance, conn *ec2.EC2) (*schema.ResourceData, error) {
	d.SetId(*instance.InstanceId)

	// Set the easy attributes
	d.Set("instance_state", instance.State.Name)
	d.Set("id", instance.InstanceId)
	d.Set("ami", instance.ImageId)
	d.Set("instance_type", instance.InstanceType)
	d.Set("key_name", instance.KeyName)
	d.Set("public_dns", instance.PublicDnsName)
	d.Set("public_ip", instance.PublicIpAddress)
	d.Set("private_dns", instance.PrivateDnsName)
	d.Set("private_ip", instance.PrivateIpAddress)
	d.Set("iam_instance_profile", iamInstanceProfileArnToName(instance.IamInstanceProfile))
	if instance.Placement != nil {
		d.Set("availability_zone", instance.Placement.AvailabilityZone)
	}
	if instance.Placement.Tenancy != nil {
		d.Set("tenancy", instance.Placement.Tenancy)
	}

	// Iterate through network interfaces, and set subnet, network_interface, public_addr
	if len(instance.NetworkInterfaces) > 0 {
		for _, ni := range instance.NetworkInterfaces {
			if *ni.Attachment.DeviceIndex == 0 {
				d.Set("subnet_id", ni.SubnetId)
				d.Set("network_interface_id", ni.NetworkInterfaceId)
				d.Set("associate_public_ip_address", ni.Association != nil)
			}
		}
	} else {
		d.Set("subnet_id", instance.SubnetId)
		d.Set("network_interface_id", "")
	}

	d.Set("ebs_optimized", instance.EbsOptimized)
	if instance.SubnetId != nil && *instance.SubnetId != "" {
		d.Set("source_dest_check", instance.SourceDestCheck)
	}

	if instance.Monitoring != nil && instance.Monitoring.State != nil {
		monitoringState := *instance.Monitoring.State
		d.Set("monitoring", monitoringState == "enabled" || monitoringState == "pending")
	}

	d.Set("tags", dataSourceTags(instance.Tags))

	// Security Groups
	if err := readSecurityGroups(d, instance); err != nil {
		return nil, err
	}

	// Block devices
	if err := readBlockDevices(d, instance, conn); err != nil {
		return nil, err
	}
	if _, ok := d.GetOk("ephemeral_block_device"); !ok {
		d.Set("ephemeral_block_device", []interface{}{})
	}

	// Look up and Set Instance Attributes
	{
		attr, err := conn.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
			Attribute:  aws.String("disableApiTermination"),
			InstanceId: aws.String(d.Id()),
		})
		if err != nil {
			return nil, err
		}
		d.Set("disable_api_termination", attr.DisableApiTermination.Value)
	}
	{
		attr, err := conn.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
			Attribute:  aws.String(ec2.InstanceAttributeNameUserData),
			InstanceId: aws.String(d.Id()),
		})
		if err != nil {
			return nil, err
		}
		if attr.UserData.Value != nil {
			d.Set("user_data", userDataHashSum(*attr.UserData.Value))
		}
	}

	return d, nil
}
