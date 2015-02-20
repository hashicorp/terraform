package aws

import (
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/autoscaling"
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
		},
	}
}

func resourceAwsLaunchConfigurationCreate(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	var createLaunchConfigurationOpts autoscaling.CreateLaunchConfigurationType
	createLaunchConfigurationOpts.LaunchConfigurationName = aws.String(d.Get("name").(string))
	createLaunchConfigurationOpts.ImageID = aws.String(d.Get("image_id").(string))
	createLaunchConfigurationOpts.InstanceType = aws.String(d.Get("instance_type").(string))

	if v, ok := d.GetOk("user_data"); ok {
		createLaunchConfigurationOpts.UserData = aws.String(base64.StdEncoding.EncodeToString([]byte(v.(string))))
	}
	if v, ok := d.GetOk("associate_public_ip_address"); ok {
		createLaunchConfigurationOpts.AssociatePublicIPAddress = aws.Boolean(v.(bool))
	}
	if v, ok := d.GetOk("iam_instance_profile"); ok {
		createLaunchConfigurationOpts.IAMInstanceProfile = aws.String(v.(string))
	}
	if v, ok := d.GetOk("key_name"); ok {
		createLaunchConfigurationOpts.KeyName = aws.String(v.(string))
	}
	if v, ok := d.GetOk("spot_price"); ok {
		createLaunchConfigurationOpts.SpotPrice = aws.String(v.(string))
	}

	if v, ok := d.GetOk("security_groups"); ok {
		createLaunchConfigurationOpts.SecurityGroups = expandStringList(
			v.(*schema.Set).List())
	}

	log.Printf("[DEBUG] autoscaling create launch configuration: %#v", createLaunchConfigurationOpts)
	err := autoscalingconn.CreateLaunchConfiguration(&createLaunchConfigurationOpts)
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

	describeOpts := autoscaling.LaunchConfigurationNamesType{
		LaunchConfigurationNames: []string{d.Id()},
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
	if *describConfs.LaunchConfigurations[0].LaunchConfigurationName != d.Id() {
		return fmt.Errorf(
			"Unable to find launch configuration: %#v",
			describConfs.LaunchConfigurations)
	}

	lc := describConfs.LaunchConfigurations[0]

	d.Set("key_name", *lc.KeyName)
	d.Set("image_id", *lc.ImageID)
	d.Set("instance_type", *lc.InstanceType)
	d.Set("name", *lc.LaunchConfigurationName)

	if lc.IAMInstanceProfile != nil {
		d.Set("iam_instance_profile", *lc.IAMInstanceProfile)
	} else {
		d.Set("iam_instance_profile", nil)
	}

	if lc.SpotPrice != nil {
		d.Set("spot_price", *lc.SpotPrice)
	} else {
		d.Set("spot_price", nil)
	}

	if lc.SecurityGroups != nil {
		d.Set("security_groups", lc.SecurityGroups)
	} else {
		d.Set("security_groups", nil)
	}
	return nil
}

func resourceAwsLaunchConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	log.Printf("[DEBUG] Launch Configuration destroy: %v", d.Id())
	err := autoscalingconn.DeleteLaunchConfiguration(
		&autoscaling.LaunchConfigurationNameType{LaunchConfigurationName: aws.String(d.Id())})
	if err != nil {
		autoscalingerr, ok := err.(aws.APIError)
		if ok && autoscalingerr.Code == "InvalidConfiguration.NotFound" {
			return nil
		}

		return err
	}

	return nil
}
