package aws

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsLaunchTemplate() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsLaunchTemplateRead,
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"default_version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"latest_version": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"block_device_mappings": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"no_device": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"virtual_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ebs": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"delete_on_termination": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"encrypted": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"iops": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"kms_key_id": {
										Type:     schema.TypeString,
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
					},
				},
			},
			"credit_specification": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cpu_credits": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"disable_api_termination": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"ebs_optimized": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"elastic_gpu_specifications": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"type": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"iam_instance_profile": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"arn": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"image_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"instance_initiated_shutdown_behavior": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"instance_market_options": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"market_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"spot_options": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"block_duration_minutes": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"instance_interruption_behavior": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"max_price": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"spot_instance_type": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"valid_until": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
			"instance_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"kernel_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"key_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"monitoring": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"enabled": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
			"network_interfaces": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"associate_public_ip_address": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"delete_on_termination": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"device_index": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"security_groups": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"ipv6_address_count": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"ipv6_addresses": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"network_interface_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"private_ip_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ipv4_addresses": {
							Type:     schema.TypeSet,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"ipv4_address_count": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"subnet_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"placement": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"affinity": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"availability_zone": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"group_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"host_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"spread_domain": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"tenancy": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"ram_disk_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"security_group_names": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"vpc_security_group_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"tag_specifications": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"resource_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"tags": tagsSchemaComputed(),
					},
				},
			},
			"user_data": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsLaunchTemplateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[DEBUG] Reading launch template %s", d.Get("name"))

	dlt, err := conn.DescribeLaunchTemplates(&ec2.DescribeLaunchTemplatesInput{
		LaunchTemplateNames: []*string{aws.String(d.Get("name").(string))},
	})

	if isAWSErr(err, ec2.LaunchTemplateErrorCodeLaunchTemplateIdDoesNotExist, "") {
		log.Printf("[WARN] launch template (%s) not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	// AWS SDK constant above is currently incorrect
	if isAWSErr(err, "InvalidLaunchTemplateId.NotFound", "") {
		log.Printf("[WARN] launch template (%s) not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("Error getting launch template: %s", err)
	}

	if dlt == nil || len(dlt.LaunchTemplates) == 0 {
		log.Printf("[WARN] launch template (%s) not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	log.Printf("[DEBUG] Found launch template %s", d.Id())

	lt := dlt.LaunchTemplates[0]
	d.SetId(*lt.LaunchTemplateId)
	d.Set("name", lt.LaunchTemplateName)
	d.Set("latest_version", lt.LatestVersionNumber)
	d.Set("default_version", lt.DefaultVersionNumber)
	d.Set("tags", tagsToMap(lt.Tags))

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "ec2",
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("launch-template/%s", d.Id()),
	}.String()
	d.Set("arn", arn)

	version := strconv.Itoa(int(*lt.LatestVersionNumber))
	dltv, err := conn.DescribeLaunchTemplateVersions(&ec2.DescribeLaunchTemplateVersionsInput{
		LaunchTemplateId: aws.String(d.Id()),
		Versions:         []*string{aws.String(version)},
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Received launch template version %q (version %d)", d.Id(), *lt.LatestVersionNumber)

	ltData := dltv.LaunchTemplateVersions[0].LaunchTemplateData

	d.Set("disable_api_termination", ltData.DisableApiTermination)
	d.Set("image_id", ltData.ImageId)
	d.Set("instance_initiated_shutdown_behavior", ltData.InstanceInitiatedShutdownBehavior)
	d.Set("instance_type", ltData.InstanceType)
	d.Set("kernel_id", ltData.KernelId)
	d.Set("key_name", ltData.KeyName)
	d.Set("ram_disk_id", ltData.RamDiskId)
	d.Set("security_group_names", aws.StringValueSlice(ltData.SecurityGroups))
	d.Set("user_data", ltData.UserData)
	d.Set("vpc_security_group_ids", aws.StringValueSlice(ltData.SecurityGroupIds))
	d.Set("ebs_optimized", "")

	if ltData.EbsOptimized != nil {
		d.Set("ebs_optimized", strconv.FormatBool(aws.BoolValue(ltData.EbsOptimized)))
	}

	if err := d.Set("block_device_mappings", getBlockDeviceMappings(ltData.BlockDeviceMappings)); err != nil {
		return err
	}

	if strings.HasPrefix(aws.StringValue(ltData.InstanceType), "t2") || strings.HasPrefix(aws.StringValue(ltData.InstanceType), "t3") {
		if err := d.Set("credit_specification", getCreditSpecification(ltData.CreditSpecification)); err != nil {
			return err
		}
	}

	if err := d.Set("elastic_gpu_specifications", getElasticGpuSpecifications(ltData.ElasticGpuSpecifications)); err != nil {
		return err
	}

	if err := d.Set("iam_instance_profile", getIamInstanceProfile(ltData.IamInstanceProfile)); err != nil {
		return err
	}

	if err := d.Set("instance_market_options", getInstanceMarketOptions(ltData.InstanceMarketOptions)); err != nil {
		return err
	}

	if err := d.Set("monitoring", getMonitoring(ltData.Monitoring)); err != nil {
		return err
	}

	if err := d.Set("network_interfaces", getNetworkInterfaces(ltData.NetworkInterfaces)); err != nil {
		return err
	}

	if err := d.Set("placement", getPlacement(ltData.Placement)); err != nil {
		return err
	}

	if err := d.Set("tag_specifications", getTagSpecifications(ltData.TagSpecifications)); err != nil {
		return err
	}

	return nil
}
