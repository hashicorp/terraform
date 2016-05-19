package aws

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLaunchConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLaunchConfigurationCreate,
		Read:   resourceAwsLaunchConfigurationRead,
		Delete: resourceAwsLaunchConfigurationDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// https://github.com/boto/botocore/blob/9f322b1/botocore/data/autoscaling/2011-01-01/service-2.json#L1932-L1939
					value := v.(string)
					if len(value) > 255 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 255 characters", k))
					}
					return
				},
			},

			"name_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, errors []error) {
					// https://github.com/boto/botocore/blob/9f322b1/botocore/data/autoscaling/2011-01-01/service-2.json#L1932-L1939
					// uuid is 26 characters, limit the prefix to 229.
					value := v.(string)
					if len(value) > 229 {
						errors = append(errors, fmt.Errorf(
							"%q cannot be longer than 229 characters, name is limited to 255", k))
					}
					return
				},
			},

			"image_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"instance_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"iam_instance_profile": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"key_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"user_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					switch v.(type) {
					case string:
						hash := sha1.Sum([]byte(v.(string)))
						return hex.EncodeToString(hash[:])
					default:
						return ""
					}
				},
			},

			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"associate_public_ip_address": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  false,
			},

			"spot_price": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"ebs_optimized": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"placement_tenancy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"enable_monitoring": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
				Default:  true,
			},

			"ebs_block_device": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delete_on_termination": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
							ForceNew: true,
						},

						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"iops": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"snapshot_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"encrypted": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["snapshot_id"].(string)))
					return hashcode.String(buf.String())
				},
			},

			"ephemeral_block_device": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},

						"virtual_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["virtual_name"].(string)))
					return hashcode.String(buf.String())
				},
			},

			"root_block_device": &schema.Schema{
				// TODO: This is a set because we don't support singleton
				//       sub-resources today. We'll enforce that the set only ever has
				//       length zero or one below. When TF gains support for
				//       sub-resources this can be converted.
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					// "You can only modify the volume size, volume type, and Delete on
					// Termination flag on the block device mapping entry for the root
					// device volume." - bit.ly/ec2bdmap
					Schema: map[string]*schema.Schema{
						"delete_on_termination": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
							ForceNew: true,
						},

						"iops": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
					},
				},
				Set: func(v interface{}) int {
					// there can be only one root device; no need to hash anything
					return 0
				},
			},
		},
	}
}

func resourceAwsLaunchConfigurationCreate(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn
	ec2conn := meta.(*AWSClient).ec2conn

	createLaunchConfigurationOpts := autoscaling.CreateLaunchConfigurationInput{
		LaunchConfigurationName: aws.String(d.Get("name").(string)),
		ImageId:                 aws.String(d.Get("image_id").(string)),
		InstanceType:            aws.String(d.Get("instance_type").(string)),
		EbsOptimized:            aws.Bool(d.Get("ebs_optimized").(bool)),
	}

	if v, ok := d.GetOk("user_data"); ok {
		userData := base64.StdEncoding.EncodeToString([]byte(v.(string)))
		createLaunchConfigurationOpts.UserData = aws.String(userData)
	}

	createLaunchConfigurationOpts.InstanceMonitoring = &autoscaling.InstanceMonitoring{
		Enabled: aws.Bool(d.Get("enable_monitoring").(bool)),
	}

	if v, ok := d.GetOk("iam_instance_profile"); ok {
		createLaunchConfigurationOpts.IamInstanceProfile = aws.String(v.(string))
	}

	if v, ok := d.GetOk("placement_tenancy"); ok {
		createLaunchConfigurationOpts.PlacementTenancy = aws.String(v.(string))
	}

	if v, ok := d.GetOk("associate_public_ip_address"); ok {
		createLaunchConfigurationOpts.AssociatePublicIpAddress = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("key_name"); ok {
		createLaunchConfigurationOpts.KeyName = aws.String(v.(string))
	}
	if v, ok := d.GetOk("spot_price"); ok {
		createLaunchConfigurationOpts.SpotPrice = aws.String(v.(string))
	}

	if v, ok := d.GetOk("security_groups"); ok {
		createLaunchConfigurationOpts.SecurityGroups = expandStringList(
			v.(*schema.Set).List(),
		)
	}

	var blockDevices []*autoscaling.BlockDeviceMapping

	if v, ok := d.GetOk("ebs_block_device"); ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			ebs := &autoscaling.Ebs{
				DeleteOnTermination: aws.Bool(bd["delete_on_termination"].(bool)),
			}

			if v, ok := bd["snapshot_id"].(string); ok && v != "" {
				ebs.SnapshotId = aws.String(v)
			}

			if v, ok := bd["encrypted"].(bool); ok && v {
				ebs.Encrypted = aws.Bool(v)
			}

			if v, ok := bd["volume_size"].(int); ok && v != 0 {
				ebs.VolumeSize = aws.Int64(int64(v))
			}

			if v, ok := bd["volume_type"].(string); ok && v != "" {
				ebs.VolumeType = aws.String(v)
			}

			if v, ok := bd["iops"].(int); ok && v > 0 {
				ebs.Iops = aws.Int64(int64(v))
			}

			blockDevices = append(blockDevices, &autoscaling.BlockDeviceMapping{
				DeviceName: aws.String(bd["device_name"].(string)),
				Ebs:        ebs,
			})
		}
	}

	if v, ok := d.GetOk("ephemeral_block_device"); ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			blockDevices = append(blockDevices, &autoscaling.BlockDeviceMapping{
				DeviceName:  aws.String(bd["device_name"].(string)),
				VirtualName: aws.String(bd["virtual_name"].(string)),
			})
		}
	}

	if v, ok := d.GetOk("root_block_device"); ok {
		vL := v.(*schema.Set).List()
		if len(vL) > 1 {
			return fmt.Errorf("Cannot specify more than one root_block_device.")
		}
		for _, v := range vL {
			bd := v.(map[string]interface{})
			ebs := &autoscaling.Ebs{
				DeleteOnTermination: aws.Bool(bd["delete_on_termination"].(bool)),
			}

			if v, ok := bd["volume_size"].(int); ok && v != 0 {
				ebs.VolumeSize = aws.Int64(int64(v))
			}

			if v, ok := bd["volume_type"].(string); ok && v != "" {
				ebs.VolumeType = aws.String(v)
			}

			if v, ok := bd["iops"].(int); ok && v > 0 {
				ebs.Iops = aws.Int64(int64(v))
			}

			if dn, err := fetchRootDeviceName(d.Get("image_id").(string), ec2conn); err == nil {
				if dn == nil {
					return fmt.Errorf(
						"Expected to find a Root Device name for AMI (%s), but got none",
						d.Get("image_id").(string))
				}
				blockDevices = append(blockDevices, &autoscaling.BlockDeviceMapping{
					DeviceName: dn,
					Ebs:        ebs,
				})
			} else {
				return err
			}
		}
	}

	if len(blockDevices) > 0 {
		createLaunchConfigurationOpts.BlockDeviceMappings = blockDevices
	}

	var lcName string
	if v, ok := d.GetOk("name"); ok {
		lcName = v.(string)
	} else if v, ok := d.GetOk("name_prefix"); ok {
		lcName = resource.PrefixedUniqueId(v.(string))
	} else {
		lcName = resource.UniqueId()
	}
	createLaunchConfigurationOpts.LaunchConfigurationName = aws.String(lcName)

	log.Printf(
		"[DEBUG] autoscaling create launch configuration: %s", createLaunchConfigurationOpts)

	// IAM profiles can take ~10 seconds to propagate in AWS:
	// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html#launch-instance-with-role-console
	err := resource.Retry(30*time.Second, func() *resource.RetryError {
		_, err := autoscalingconn.CreateLaunchConfiguration(&createLaunchConfigurationOpts)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if strings.Contains(awsErr.Message(), "Invalid IamInstanceProfile") {
					return resource.RetryableError(err)
				}
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("Error creating launch configuration: %s", err)
	}

	d.SetId(lcName)
	log.Printf("[INFO] launch configuration ID: %s", d.Id())

	// We put a Retry here since sometimes eventual consistency bites
	// us and we need to retry a few times to get the LC to load properly
	return resource.Retry(30*time.Second, func() *resource.RetryError {
		err := resourceAwsLaunchConfigurationRead(d, meta)
		if err != nil {
			return resource.RetryableError(err)
		}
		return nil
	})
}

func resourceAwsLaunchConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn
	ec2conn := meta.(*AWSClient).ec2conn

	describeOpts := autoscaling.DescribeLaunchConfigurationsInput{
		LaunchConfigurationNames: []*string{aws.String(d.Id())},
	}

	log.Printf("[DEBUG] launch configuration describe configuration: %s", describeOpts)
	describConfs, err := autoscalingconn.DescribeLaunchConfigurations(&describeOpts)
	if err != nil {
		return fmt.Errorf("Error retrieving launch configuration: %s", err)
	}
	if len(describConfs.LaunchConfigurations) == 0 {
		d.SetId("")
		return nil
	}

	// Verify AWS returned our launch configuration
	if *describConfs.LaunchConfigurations[0].LaunchConfigurationName != d.Id() {
		return fmt.Errorf(
			"Unable to find launch configuration: %#v",
			describConfs.LaunchConfigurations)
	}

	lc := describConfs.LaunchConfigurations[0]

	d.Set("key_name", lc.KeyName)
	d.Set("image_id", lc.ImageId)
	d.Set("instance_type", lc.InstanceType)
	d.Set("name", lc.LaunchConfigurationName)

	d.Set("iam_instance_profile", lc.IamInstanceProfile)
	d.Set("ebs_optimized", lc.EbsOptimized)
	d.Set("spot_price", lc.SpotPrice)
	d.Set("enable_monitoring", lc.InstanceMonitoring.Enabled)
	d.Set("security_groups", lc.SecurityGroups)

	if err := readLCBlockDevices(d, lc, ec2conn); err != nil {
		return err
	}

	return nil
}

func resourceAwsLaunchConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	log.Printf("[DEBUG] Launch Configuration destroy: %v", d.Id())
	_, err := autoscalingconn.DeleteLaunchConfiguration(
		&autoscaling.DeleteLaunchConfigurationInput{
			LaunchConfigurationName: aws.String(d.Id()),
		})
	if err != nil {
		autoscalingerr, ok := err.(awserr.Error)
		if ok && (autoscalingerr.Code() == "InvalidConfiguration.NotFound" || autoscalingerr.Code() == "ValidationError") {
			log.Printf("[DEBUG] Launch configuration (%s) not found", d.Id())
			return nil
		}

		return err
	}

	return nil
}

func readLCBlockDevices(d *schema.ResourceData, lc *autoscaling.LaunchConfiguration, ec2conn *ec2.EC2) error {
	ibds, err := readBlockDevicesFromLaunchConfiguration(d, lc, ec2conn)
	if err != nil {
		return err
	}

	if err := d.Set("ebs_block_device", ibds["ebs"]); err != nil {
		return err
	}
	if err := d.Set("ephemeral_block_device", ibds["ephemeral"]); err != nil {
		return err
	}
	if ibds["root"] != nil {
		if err := d.Set("root_block_device", []interface{}{ibds["root"]}); err != nil {
			return err
		}
	} else {
		d.Set("root_block_device", []interface{}{})
	}

	return nil
}

func readBlockDevicesFromLaunchConfiguration(d *schema.ResourceData, lc *autoscaling.LaunchConfiguration, ec2conn *ec2.EC2) (
	map[string]interface{}, error) {
	blockDevices := make(map[string]interface{})
	blockDevices["ebs"] = make([]map[string]interface{}, 0)
	blockDevices["ephemeral"] = make([]map[string]interface{}, 0)
	blockDevices["root"] = nil
	if len(lc.BlockDeviceMappings) == 0 {
		return nil, nil
	}
	rootDeviceName, err := fetchRootDeviceName(d.Get("image_id").(string), ec2conn)
	if err != nil {
		return nil, err
	}
	if rootDeviceName == nil {
		// We do this so the value is empty so we don't have to do nil checks later
		var blank string
		rootDeviceName = &blank
	}
	for _, bdm := range lc.BlockDeviceMappings {
		bd := make(map[string]interface{})
		if bdm.Ebs != nil && bdm.Ebs.DeleteOnTermination != nil {
			bd["delete_on_termination"] = *bdm.Ebs.DeleteOnTermination
		}
		if bdm.Ebs != nil && bdm.Ebs.VolumeSize != nil {
			bd["volume_size"] = *bdm.Ebs.VolumeSize
		}
		if bdm.Ebs != nil && bdm.Ebs.VolumeType != nil {
			bd["volume_type"] = *bdm.Ebs.VolumeType
		}
		if bdm.Ebs != nil && bdm.Ebs.Iops != nil {
			bd["iops"] = *bdm.Ebs.Iops
		}
		if bdm.Ebs != nil && bdm.Ebs.Encrypted != nil {
			bd["encrypted"] = *bdm.Ebs.Encrypted
		}
		if bdm.DeviceName != nil && *bdm.DeviceName == *rootDeviceName {
			blockDevices["root"] = bd
		} else {
			if bdm.DeviceName != nil {
				bd["device_name"] = *bdm.DeviceName
			}
			if bdm.VirtualName != nil {
				bd["virtual_name"] = *bdm.VirtualName
				blockDevices["ephemeral"] = append(blockDevices["ephemeral"].([]map[string]interface{}), bd)
			} else {
				if bdm.Ebs != nil && bdm.Ebs.SnapshotId != nil {
					bd["snapshot_id"] = *bdm.Ebs.SnapshotId
				}
				blockDevices["ebs"] = append(blockDevices["ebs"].([]map[string]interface{}), bd)
			}
		}
	}
	return blockDevices, nil
}
