package aws

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/autoscaling"
)

func resourceAwsLaunchConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLaunchConfigurationCreate,
		Read:   resourceAwsLaunchConfigurationRead,
		Delete: resourceAwsLaunchConfigurationDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
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
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"associate_public_ip_address": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"spot_price": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"block_device": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"virtual_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"snapshot_id": &schema.Schema{
							Type:     schema.TypeString,
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

						"volume_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"delete_on_termination": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
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
				Set: resourceAwsInstanceBlockDevicesHash,
			},
		},
	}
}

func resourceAwsLaunchConfigurationCreate(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	var createLaunchConfigurationOpts autoscaling.CreateLaunchConfiguration
	createLaunchConfigurationOpts.Name = d.Get("name").(string)
	createLaunchConfigurationOpts.IamInstanceProfile = d.Get("iam_instance_profile").(string)
	createLaunchConfigurationOpts.ImageId = d.Get("image_id").(string)
	createLaunchConfigurationOpts.InstanceType = d.Get("instance_type").(string)
	createLaunchConfigurationOpts.KeyName = d.Get("key_name").(string)
	createLaunchConfigurationOpts.UserData = d.Get("user_data").(string)
	createLaunchConfigurationOpts.AssociatePublicIpAddress = d.Get("associate_public_ip_address").(bool)
	createLaunchConfigurationOpts.SpotPrice = d.Get("spot_price").(string)

	if v, ok := d.GetOk("security_groups"); ok {
		createLaunchConfigurationOpts.SecurityGroups = expandStringList(
			v.(*schema.Set).List())
	}

	if v := d.Get("block_device"); v != nil {
		vs := v.(*schema.Set).List()
		if len(vs) > 0 {
			createLaunchConfigurationOpts.BlockDevices = make([]autoscaling.BlockDeviceMapping, len(vs))
			for i, v := range vs {
				bd := v.(map[string]interface{})
				createLaunchConfigurationOpts.BlockDevices[i].DeviceName = bd["device_name"].(string)
				createLaunchConfigurationOpts.BlockDevices[i].VirtualName = bd["virtual_name"].(string)
				createLaunchConfigurationOpts.BlockDevices[i].SnapshotId = bd["snapshot_id"].(string)
				createLaunchConfigurationOpts.BlockDevices[i].VolumeType = bd["volume_type"].(string)
				createLaunchConfigurationOpts.BlockDevices[i].VolumeSize = int64(bd["volume_size"].(int))
				createLaunchConfigurationOpts.BlockDevices[i].DeleteOnTermination = bd["delete_on_termination"].(bool)
				createLaunchConfigurationOpts.BlockDevices[i].Encrypted = bd["encrypted"].(bool)
			}
		}
	}

	log.Printf("[DEBUG] autoscaling create launch configuration: %#v", createLaunchConfigurationOpts)
	_, err := autoscalingconn.CreateLaunchConfiguration(&createLaunchConfigurationOpts)
	if err != nil {
		return fmt.Errorf("Error creating launch configuration: %s", err)
	}

	d.SetId(d.Get("name").(string))
	log.Printf("[INFO] launch configuration ID: %s", d.Id())

	// We put a Retry here since sometimes eventual consistency bites
	// us and we need to retry a few times to get the LC to load properly
	return resource.Retry(30*time.Second, func() error {
		return resourceAwsLaunchConfigurationRead(d, meta)
	})
}

func resourceAwsLaunchConfigurationRead(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	describeOpts := autoscaling.DescribeLaunchConfigurations{
		Names: []string{d.Id()},
	}

	log.Printf("[DEBUG] launch configuration describe configuration: %#v", describeOpts)
	describConfs, err := autoscalingconn.DescribeLaunchConfigurations(&describeOpts)
	if err != nil {
		return fmt.Errorf("Error retrieving launch configuration: %s", err)
	}
	if len(describConfs.LaunchConfigurations) == 0 {
		d.SetId("")
		return nil
	}

	// Verify AWS returned our launch configuration
	if describConfs.LaunchConfigurations[0].Name != d.Id() {
		return fmt.Errorf(
			"Unable to find launch configuration: %#v",
			describConfs.LaunchConfigurations)
	}

	lc := describConfs.LaunchConfigurations[0]

	d.Set("key_name", lc.KeyName)
	d.Set("iam_instance_profile", lc.IamInstanceProfile)
	d.Set("image_id", lc.ImageId)
	d.Set("instance_type", lc.InstanceType)
	d.Set("name", lc.Name)
	d.Set("security_groups", lc.SecurityGroups)
	d.Set("spot_price", lc.SpotPrice)

	bds := make([]map[string]interface{}, len(lc.BlockDevices))
	for i, m := range lc.BlockDevices {
		bds[i] = make(map[string]interface{})
		bds[i]["device_name"] = m.DeviceName
		bds[i]["snapshot_id"] = m.SnapshotId
		bds[i]["volume_type"] = m.VolumeType
		bds[i]["volume_size"] = m.VolumeSize
		bds[i]["delete_on_termination"] = m.DeleteOnTermination
		bds[i]["encrypted"] = m.Encrypted
	}
	d.Set("block_device", bds)

	return nil
}

func resourceAwsLaunchConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	log.Printf("[DEBUG] Launch Configuration destroy: %v", d.Id())
	_, err := autoscalingconn.DeleteLaunchConfiguration(
		&autoscaling.DeleteLaunchConfiguration{Name: d.Id()})
	if err != nil {
		autoscalingerr, ok := err.(*autoscaling.Error)
		if ok && autoscalingerr.Code == "InvalidConfiguration.NotFound" {
			return nil
		}

		return err
	}

	return nil
}
