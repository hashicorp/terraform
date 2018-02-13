package aws

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsInstanceCreate,
		Read:   resourceAwsInstanceRead,
		Update: resourceAwsInstanceUpdate,
		Delete: resourceAwsInstanceDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		SchemaVersion: 1,
		MigrateState:  resourceAwsInstanceMigrateState,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"ami": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"associate_public_ip_address": {
				Type:     schema.TypeBool,
				ForceNew: true,
				Computed: true,
				Optional: true,
			},

			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"placement_group": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"instance_type": {
				Type:     schema.TypeString,
				Required: true,
			},

			"key_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"private_ip": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"source_dest_check": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Suppress diff if network_interface is set
					_, ok := d.GetOk("network_interface")
					return ok
				},
			},

			"user_data": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"user_data_base64"},
				StateFunc: func(v interface{}) string {
					switch v.(type) {
					case string:
						return userDataHashSum(v.(string))
					default:
						return ""
					}
				},
				ValidateFunc: validateInstanceUserDataSize,
			},

			"user_data_base64": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"user_data"},
				ValidateFunc: func(v interface{}, name string) (warns []string, errs []error) {
					s := v.(string)
					if !isBase64Encoded([]byte(s)) {
						errs = append(errs, fmt.Errorf(
							"%s: must be base64-encoded", name,
						))
					}
					return
				},
			},

			"security_groups": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"vpc_security_group_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"public_dns": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// TODO: Deprecate me v0.10.0
			"network_interface_id": {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Please use `primary_network_interface_id` instead",
			},

			"primary_network_interface_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"network_interface": {
				ConflictsWith: []string{"associate_public_ip_address", "subnet_id", "private_ip", "vpc_security_group_ids", "security_groups", "ipv6_addresses", "ipv6_address_count", "source_dest_check"},
				Type:          schema.TypeSet,
				Optional:      true,
				Computed:      true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delete_on_termination": {
							Type:     schema.TypeBool,
							Default:  false,
							Optional: true,
							ForceNew: true,
						},
						"network_interface_id": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"device_index": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
					},
				},
			},

			"public_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"instance_state": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"private_dns": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ebs_optimized": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"disable_api_termination": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"instance_initiated_shutdown_behavior": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"monitoring": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			"iam_instance_profile": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"ipv6_address_count": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"ipv6_addresses": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},

			"tenancy": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"tags": tagsSchema(),

			"volume_tags": tagsSchemaComputed(),

			"block_device": {
				Type:     schema.TypeMap,
				Optional: true,
				Removed:  "Split out into three sub-types; see Changelog and Docs",
			},

			"ebs_block_device": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delete_on_termination": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
							ForceNew: true,
						},

						"device_name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"encrypted": {
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"iops": {
							Type:             schema.TypeInt,
							Optional:         true,
							Computed:         true,
							ForceNew:         true,
							DiffSuppressFunc: iopsDiffSuppressFunc,
						},

						"snapshot_id": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_size": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_type": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_id": {
							Type:     schema.TypeString,
							Computed: true,
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

			"ephemeral_block_device": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
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
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["virtual_name"].(string)))
					if v, ok := m["no_device"].(bool); ok && v {
						buf.WriteString(fmt.Sprintf("%t-", v))
					}
					return hashcode.String(buf.String())
				},
			},

			"root_block_device": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					// "You can only modify the volume size, volume type, and Delete on
					// Termination flag on the block device mapping entry for the root
					// device volume." - bit.ly/ec2bdmap
					Schema: map[string]*schema.Schema{
						"delete_on_termination": {
							Type:     schema.TypeBool,
							Optional: true,
							Default:  true,
							ForceNew: true,
						},

						"iops": {
							Type:             schema.TypeInt,
							Optional:         true,
							Computed:         true,
							ForceNew:         true,
							DiffSuppressFunc: iopsDiffSuppressFunc,
						},

						"volume_size": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_type": {
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},

						"volume_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func iopsDiffSuppressFunc(k, old, new string, d *schema.ResourceData) bool {
	// Suppress diff if volume_type is not io1
	i := strings.LastIndexByte(k, '.')
	vt := k[:i+1] + "volume_type"
	v := d.Get(vt).(string)
	return strings.ToLower(v) != ec2.VolumeTypeIo1
}

func resourceAwsInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	instanceOpts, err := buildAwsInstanceOpts(d, meta)
	if err != nil {
		return err
	}

	// Build the creation struct
	runOpts := &ec2.RunInstancesInput{
		BlockDeviceMappings:   instanceOpts.BlockDeviceMappings,
		DisableApiTermination: instanceOpts.DisableAPITermination,
		EbsOptimized:          instanceOpts.EBSOptimized,
		Monitoring:            instanceOpts.Monitoring,
		IamInstanceProfile:    instanceOpts.IAMInstanceProfile,
		ImageId:               instanceOpts.ImageID,
		InstanceInitiatedShutdownBehavior: instanceOpts.InstanceInitiatedShutdownBehavior,
		InstanceType:                      instanceOpts.InstanceType,
		Ipv6AddressCount:                  instanceOpts.Ipv6AddressCount,
		Ipv6Addresses:                     instanceOpts.Ipv6Addresses,
		KeyName:                           instanceOpts.KeyName,
		MaxCount:                          aws.Int64(int64(1)),
		MinCount:                          aws.Int64(int64(1)),
		NetworkInterfaces:                 instanceOpts.NetworkInterfaces,
		Placement:                         instanceOpts.Placement,
		PrivateIpAddress:                  instanceOpts.PrivateIPAddress,
		SecurityGroupIds:                  instanceOpts.SecurityGroupIDs,
		SecurityGroups:                    instanceOpts.SecurityGroups,
		SubnetId:                          instanceOpts.SubnetID,
		UserData:                          instanceOpts.UserData64,
	}

	_, ipv6CountOk := d.GetOk("ipv6_address_count")
	_, ipv6AddressOk := d.GetOk("ipv6_addresses")

	if ipv6AddressOk && ipv6CountOk {
		return fmt.Errorf("Only 1 of `ipv6_address_count` or `ipv6_addresses` can be specified")
	}

	restricted := meta.(*AWSClient).IsGovCloud() || meta.(*AWSClient).IsChinaCloud()
	if !restricted {
		tagsSpec := make([]*ec2.TagSpecification, 0)

		if v, ok := d.GetOk("tags"); ok {
			tags := tagsFromMap(v.(map[string]interface{}))

			spec := &ec2.TagSpecification{
				ResourceType: aws.String("instance"),
				Tags:         tags,
			}

			tagsSpec = append(tagsSpec, spec)
		}

		if v, ok := d.GetOk("volume_tags"); ok {
			tags := tagsFromMap(v.(map[string]interface{}))

			spec := &ec2.TagSpecification{
				ResourceType: aws.String("volume"),
				Tags:         tags,
			}

			tagsSpec = append(tagsSpec, spec)
		}

		if len(tagsSpec) > 0 {
			runOpts.TagSpecifications = tagsSpec
		}
	}

	// Create the instance
	log.Printf("[DEBUG] Run configuration: %s", runOpts)

	var runResp *ec2.Reservation
	err = resource.Retry(30*time.Second, func() *resource.RetryError {
		var err error
		runResp, err = conn.RunInstances(runOpts)
		// IAM instance profiles can take ~10 seconds to propagate in AWS:
		// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/iam-roles-for-amazon-ec2.html#launch-instance-with-role-console
		if isAWSErr(err, "InvalidParameterValue", "Invalid IAM Instance Profile") {
			log.Print("[DEBUG] Invalid IAM Instance Profile referenced, retrying...")
			return resource.RetryableError(err)
		}
		// IAM roles can also take time to propagate in AWS:
		if isAWSErr(err, "InvalidParameterValue", " has no associated IAM Roles") {
			log.Print("[DEBUG] IAM Instance Profile appears to have no IAM roles, retrying...")
			return resource.RetryableError(err)
		}
		return resource.NonRetryableError(err)
	})
	// Warn if the AWS Error involves group ids, to help identify situation
	// where a user uses group ids in security_groups for the Default VPC.
	//   See https://github.com/hashicorp/terraform/issues/3798
	if isAWSErr(err, "InvalidParameterValue", "groupId is invalid") {
		return fmt.Errorf("Error launching instance, possible mismatch of Security Group IDs and Names. See AWS Instance docs here: %s.\n\n\tAWS Error: %s", "https://terraform.io/docs/providers/aws/r/instance.html", err.(awserr.Error).Message())
	}
	if err != nil {
		return fmt.Errorf("Error launching source instance: %s", err)
	}
	if runResp == nil || len(runResp.Instances) == 0 {
		return errors.New("Error launching source instance: no instances returned in response")
	}

	instance := runResp.Instances[0]
	log.Printf("[INFO] Instance ID: %s", *instance.InstanceId)

	// Store the resulting ID so we can look this up later
	d.SetId(*instance.InstanceId)

	// Wait for the instance to become running so we can get some attributes
	// that aren't available until later.
	log.Printf(
		"[DEBUG] Waiting for instance (%s) to become running",
		*instance.InstanceId)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     []string{"running"},
		Refresh:    InstanceStateRefreshFunc(conn, *instance.InstanceId, "terminated"),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	instanceRaw, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for instance (%s) to become ready: %s",
			*instance.InstanceId, err)
	}

	instance = instanceRaw.(*ec2.Instance)

	// Initialize the connection info
	if instance.PublicIpAddress != nil {
		d.SetConnInfo(map[string]string{
			"type": "ssh",
			"host": *instance.PublicIpAddress,
		})
	} else if instance.PrivateIpAddress != nil {
		d.SetConnInfo(map[string]string{
			"type": "ssh",
			"host": *instance.PrivateIpAddress,
		})
	}

	// Update if we need to
	return resourceAwsInstanceUpdate(d, meta)
}

func resourceAwsInstanceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	resp, err := conn.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(d.Id())},
	})
	if err != nil {
		// If the instance was not found, return nil so that we can show
		// that the instance is gone.
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidInstanceID.NotFound" {
			d.SetId("")
			return nil
		}

		// Some other error, report it
		return err
	}

	// If nothing was found, then return no state
	if len(resp.Reservations) == 0 {
		d.SetId("")
		return nil
	}

	instance := resp.Reservations[0].Instances[0]

	if instance.State != nil {
		// If the instance is terminated, then it is gone
		if *instance.State.Name == "terminated" {
			d.SetId("")
			return nil
		}

		d.Set("instance_state", instance.State.Name)
	}

	if instance.Placement != nil {
		d.Set("availability_zone", instance.Placement.AvailabilityZone)
	}
	if instance.Placement.GroupName != nil {
		d.Set("placement_group", instance.Placement.GroupName)
	}
	if instance.Placement.Tenancy != nil {
		d.Set("tenancy", instance.Placement.Tenancy)
	}

	d.Set("ami", instance.ImageId)
	d.Set("instance_type", instance.InstanceType)
	d.Set("key_name", instance.KeyName)
	d.Set("public_dns", instance.PublicDnsName)
	d.Set("public_ip", instance.PublicIpAddress)
	d.Set("private_dns", instance.PrivateDnsName)
	d.Set("private_ip", instance.PrivateIpAddress)
	d.Set("iam_instance_profile", iamInstanceProfileArnToName(instance.IamInstanceProfile))

	// Set configured Network Interface Device Index Slice
	// We only want to read, and populate state for the configured network_interface attachments. Otherwise, other
	// resources have the potential to attach network interfaces to the instance, and cause a perpetual create/destroy
	// diff. We should only read on changes configured for this specific resource because of this.
	var configuredDeviceIndexes []int
	if v, ok := d.GetOk("network_interface"); ok {
		vL := v.(*schema.Set).List()
		for _, vi := range vL {
			mVi := vi.(map[string]interface{})
			configuredDeviceIndexes = append(configuredDeviceIndexes, mVi["device_index"].(int))
		}
	}

	var ipv6Addresses []string
	if len(instance.NetworkInterfaces) > 0 {
		var primaryNetworkInterface ec2.InstanceNetworkInterface
		var networkInterfaces []map[string]interface{}
		for _, iNi := range instance.NetworkInterfaces {
			ni := make(map[string]interface{})
			if *iNi.Attachment.DeviceIndex == 0 {
				primaryNetworkInterface = *iNi
			}
			// If the attached network device is inside our configuration, refresh state with values found.
			// Otherwise, assume the network device was attached via an outside resource.
			for _, index := range configuredDeviceIndexes {
				if index == int(*iNi.Attachment.DeviceIndex) {
					ni["device_index"] = *iNi.Attachment.DeviceIndex
					ni["network_interface_id"] = *iNi.NetworkInterfaceId
					ni["delete_on_termination"] = *iNi.Attachment.DeleteOnTermination
				}
			}
			// Don't add empty network interfaces to schema
			if len(ni) == 0 {
				continue
			}
			networkInterfaces = append(networkInterfaces, ni)
		}
		if err := d.Set("network_interface", networkInterfaces); err != nil {
			return fmt.Errorf("Error setting network_interfaces: %v", err)
		}

		// Set primary network interface details
		// If an instance is shutting down, network interfaces are detached, and attributes may be nil,
		// need to protect against nil pointer dereferences
		if primaryNetworkInterface.SubnetId != nil {
			d.Set("subnet_id", primaryNetworkInterface.SubnetId)
		}
		if primaryNetworkInterface.NetworkInterfaceId != nil {
			d.Set("network_interface_id", primaryNetworkInterface.NetworkInterfaceId) // TODO: Deprecate me v0.10.0
			d.Set("primary_network_interface_id", primaryNetworkInterface.NetworkInterfaceId)
		}
		if primaryNetworkInterface.Ipv6Addresses != nil {
			d.Set("ipv6_address_count", len(primaryNetworkInterface.Ipv6Addresses))
		}
		if primaryNetworkInterface.SourceDestCheck != nil {
			d.Set("source_dest_check", primaryNetworkInterface.SourceDestCheck)
		}

		d.Set("associate_public_ip_address", primaryNetworkInterface.Association != nil)

		for _, address := range primaryNetworkInterface.Ipv6Addresses {
			ipv6Addresses = append(ipv6Addresses, *address.Ipv6Address)
		}

	} else {
		d.Set("subnet_id", instance.SubnetId)
		d.Set("network_interface_id", "") // TODO: Deprecate me v0.10.0
		d.Set("primary_network_interface_id", "")
	}

	if err := d.Set("ipv6_addresses", ipv6Addresses); err != nil {
		log.Printf("[WARN] Error setting ipv6_addresses for AWS Instance (%s): %s", d.Id(), err)
	}

	d.Set("ebs_optimized", instance.EbsOptimized)
	if instance.SubnetId != nil && *instance.SubnetId != "" {
		d.Set("source_dest_check", instance.SourceDestCheck)
	}

	if instance.Monitoring != nil && instance.Monitoring.State != nil {
		monitoringState := *instance.Monitoring.State
		d.Set("monitoring", monitoringState == "enabled" || monitoringState == "pending")
	}

	d.Set("tags", tagsToMap(instance.Tags))

	if err := readVolumeTags(conn, d); err != nil {
		return err
	}

	if err := readSecurityGroups(d, instance, conn); err != nil {
		return err
	}

	if err := readBlockDevices(d, instance, conn); err != nil {
		return err
	}
	if _, ok := d.GetOk("ephemeral_block_device"); !ok {
		d.Set("ephemeral_block_device", []interface{}{})
	}

	// Instance attributes
	{
		attr, err := conn.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
			Attribute:  aws.String("disableApiTermination"),
			InstanceId: aws.String(d.Id()),
		})
		if err != nil {
			return err
		}
		d.Set("disable_api_termination", attr.DisableApiTermination.Value)
	}
	{
		attr, err := conn.DescribeInstanceAttribute(&ec2.DescribeInstanceAttributeInput{
			Attribute:  aws.String(ec2.InstanceAttributeNameUserData),
			InstanceId: aws.String(d.Id()),
		})
		if err != nil {
			return err
		}
		if attr.UserData != nil && attr.UserData.Value != nil {
			// Since user_data and user_data_base64 conflict with each other,
			// we'll only set one or the other here to avoid a perma-diff.
			// Since user_data_base64 was added later, we'll prefer to set
			// user_data.
			_, b64 := d.GetOk("user_data_base64")
			if b64 {
				d.Set("user_data_base64", attr.UserData.Value)
			} else {
				d.Set("user_data", userDataHashSum(*attr.UserData.Value))
			}
		}
	}

	return nil
}

func resourceAwsInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	d.Partial(true)

	restricted := meta.(*AWSClient).IsGovCloud() || meta.(*AWSClient).IsChinaCloud()

	if d.HasChange("tags") {
		if !d.IsNewResource() || restricted {
			if err := setTags(conn, d); err != nil {
				return err
			} else {
				d.SetPartial("tags")
			}
		}
	}
	if d.HasChange("volume_tags") {
		if !d.IsNewResource() || !restricted {
			if err := setVolumeTags(conn, d); err != nil {
				return err
			} else {
				d.SetPartial("volume_tags")
			}
		}
	}

	if d.HasChange("iam_instance_profile") && !d.IsNewResource() {
		request := &ec2.DescribeIamInstanceProfileAssociationsInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("instance-id"),
					Values: []*string{aws.String(d.Id())},
				},
			},
		}

		resp, err := conn.DescribeIamInstanceProfileAssociations(request)
		if err != nil {
			return err
		}

		// An Iam Instance Profile has been provided and is pending a change
		// This means it is an association or a replacement to an association
		if _, ok := d.GetOk("iam_instance_profile"); ok {
			// Does not have an Iam Instance Profile associated with it, need to associate
			if len(resp.IamInstanceProfileAssociations) == 0 {
				err := resource.Retry(1*time.Minute, func() *resource.RetryError {
					_, err := conn.AssociateIamInstanceProfile(&ec2.AssociateIamInstanceProfileInput{
						InstanceId: aws.String(d.Id()),
						IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
							Name: aws.String(d.Get("iam_instance_profile").(string)),
						},
					})
					if err != nil {
						if isAWSErr(err, "InvalidParameterValue", "Invalid IAM Instance Profile") {
							return resource.RetryableError(err)
						}
						return resource.NonRetryableError(err)
					}
					return nil
				})
				if err != nil {
					return err
				}

			} else {
				// Has an Iam Instance Profile associated with it, need to replace the association
				associationId := resp.IamInstanceProfileAssociations[0].AssociationId

				err := resource.Retry(1*time.Minute, func() *resource.RetryError {
					_, err := conn.ReplaceIamInstanceProfileAssociation(&ec2.ReplaceIamInstanceProfileAssociationInput{
						AssociationId: associationId,
						IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
							Name: aws.String(d.Get("iam_instance_profile").(string)),
						},
					})
					if err != nil {
						if isAWSErr(err, "InvalidParameterValue", "Invalid IAM Instance Profile") {
							return resource.RetryableError(err)
						}
						return resource.NonRetryableError(err)
					}
					return nil
				})
				if err != nil {
					return err
				}
			}
			// An Iam Instance Profile has _not_ been provided but is pending a change. This means there is a pending removal
		} else {
			if len(resp.IamInstanceProfileAssociations) > 0 {
				// Has an Iam Instance Profile associated with it, need to remove the association
				associationId := resp.IamInstanceProfileAssociations[0].AssociationId

				_, err := conn.DisassociateIamInstanceProfile(&ec2.DisassociateIamInstanceProfileInput{
					AssociationId: associationId,
				})
				if err != nil {
					return err
				}
			}
		}
	}

	// SourceDestCheck can only be modified on an instance without manually specified network interfaces.
	// SourceDestCheck, in that case, is configured at the network interface level
	if _, ok := d.GetOk("network_interface"); !ok {

		// If we have a new resource and source_dest_check is still true, don't modify
		sourceDestCheck := d.Get("source_dest_check").(bool)

		// Because we're calling Update prior to Read, and the default value of `source_dest_check` is `true`,
		// HasChange() thinks there is a diff between what is set on the instance and what is set in state. We need to ensure that
		// if a diff has occured, it's not because it's a new instance.
		if d.HasChange("source_dest_check") && !d.IsNewResource() || d.IsNewResource() && !sourceDestCheck {
			// SourceDestCheck can only be set on VPC instances
			// AWS will return an error of InvalidParameterCombination if we attempt
			// to modify the source_dest_check of an instance in EC2 Classic
			log.Printf("[INFO] Modifying `source_dest_check` on Instance %s", d.Id())
			_, err := conn.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
				InstanceId: aws.String(d.Id()),
				SourceDestCheck: &ec2.AttributeBooleanValue{
					Value: aws.Bool(sourceDestCheck),
				},
			})
			if err != nil {
				if ec2err, ok := err.(awserr.Error); ok {
					// Tolerate InvalidParameterCombination error in Classic, otherwise
					// return the error
					if "InvalidParameterCombination" != ec2err.Code() {
						return err
					}
					log.Printf("[WARN] Attempted to modify SourceDestCheck on non VPC instance: %s", ec2err.Message())
				}
			}
		}
	}

	if d.HasChange("vpc_security_group_ids") {
		var groups []*string
		if v := d.Get("vpc_security_group_ids").(*schema.Set); v.Len() > 0 {
			for _, v := range v.List() {
				groups = append(groups, aws.String(v.(string)))
			}
		}
		// If a user has multiple network interface attachments on the target EC2 instance, simply modifying the
		// instance attributes via a `ModifyInstanceAttributes()` request would fail with the following error message:
		// "There are multiple interfaces attached to instance 'i-XX'. Please specify an interface ID for the operation instead."
		// Thus, we need to actually modify the primary network interface for the new security groups, as the primary
		// network interface is where we modify/create security group assignments during Create.
		log.Printf("[INFO] Modifying `vpc_security_group_ids` on Instance %q", d.Id())
		instances, err := conn.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIds: []*string{aws.String(d.Id())},
		})
		if err != nil {
			return err
		}
		instance := instances.Reservations[0].Instances[0]
		var primaryInterface ec2.InstanceNetworkInterface
		for _, ni := range instance.NetworkInterfaces {
			if *ni.Attachment.DeviceIndex == 0 {
				primaryInterface = *ni
			}
		}

		if primaryInterface.NetworkInterfaceId == nil {
			log.Print("[Error] Attempted to set vpc_security_group_ids on an instance without a primary network interface")
			return fmt.Errorf(
				"Failed to update vpc_security_group_ids on %q, which does not contain a primary network interface",
				d.Id())
		}

		if _, err := conn.ModifyNetworkInterfaceAttribute(&ec2.ModifyNetworkInterfaceAttributeInput{
			NetworkInterfaceId: primaryInterface.NetworkInterfaceId,
			Groups:             groups,
		}); err != nil {
			return err
		}
	}

	if d.HasChange("instance_type") && !d.IsNewResource() {
		log.Printf("[INFO] Stopping Instance %q for instance_type change", d.Id())
		_, err := conn.StopInstances(&ec2.StopInstancesInput{
			InstanceIds: []*string{aws.String(d.Id())},
		})

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"pending", "running", "shutting-down", "stopped", "stopping"},
			Target:     []string{"stopped"},
			Refresh:    InstanceStateRefreshFunc(conn, d.Id(), ""),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf(
				"Error waiting for instance (%s) to stop: %s", d.Id(), err)
		}

		log.Printf("[INFO] Modifying instance type %s", d.Id())
		_, err = conn.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
			InstanceId: aws.String(d.Id()),
			InstanceType: &ec2.AttributeValue{
				Value: aws.String(d.Get("instance_type").(string)),
			},
		})
		if err != nil {
			return err
		}

		log.Printf("[INFO] Starting Instance %q after instance_type change", d.Id())
		_, err = conn.StartInstances(&ec2.StartInstancesInput{
			InstanceIds: []*string{aws.String(d.Id())},
		})

		stateConf = &resource.StateChangeConf{
			Pending:    []string{"pending", "stopped"},
			Target:     []string{"running"},
			Refresh:    InstanceStateRefreshFunc(conn, d.Id(), "terminated"),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf(
				"Error waiting for instance (%s) to become ready: %s",
				d.Id(), err)
		}
	}

	if d.HasChange("disable_api_termination") {
		_, err := conn.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
			InstanceId: aws.String(d.Id()),
			DisableApiTermination: &ec2.AttributeBooleanValue{
				Value: aws.Bool(d.Get("disable_api_termination").(bool)),
			},
		})
		if err != nil {
			return err
		}
	}

	if d.HasChange("instance_initiated_shutdown_behavior") {
		log.Printf("[INFO] Modifying instance %s", d.Id())
		_, err := conn.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeInput{
			InstanceId: aws.String(d.Id()),
			InstanceInitiatedShutdownBehavior: &ec2.AttributeValue{
				Value: aws.String(d.Get("instance_initiated_shutdown_behavior").(string)),
			},
		})
		if err != nil {
			return err
		}
	}

	if d.HasChange("monitoring") {
		var mErr error
		if d.Get("monitoring").(bool) {
			log.Printf("[DEBUG] Enabling monitoring for Instance (%s)", d.Id())
			_, mErr = conn.MonitorInstances(&ec2.MonitorInstancesInput{
				InstanceIds: []*string{aws.String(d.Id())},
			})
		} else {
			log.Printf("[DEBUG] Disabling monitoring for Instance (%s)", d.Id())
			_, mErr = conn.UnmonitorInstances(&ec2.UnmonitorInstancesInput{
				InstanceIds: []*string{aws.String(d.Id())},
			})
		}
		if mErr != nil {
			return fmt.Errorf("[WARN] Error updating Instance monitoring: %s", mErr)
		}
	}

	// TODO(mitchellh): wait for the attributes we modified to
	// persist the change...

	d.Partial(false)

	return resourceAwsInstanceRead(d, meta)
}

func resourceAwsInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	if err := awsTerminateInstance(conn, d.Id(), d); err != nil {
		return err
	}

	d.SetId("")
	return nil
}

// InstanceStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an EC2 instance.
func InstanceStateRefreshFunc(conn *ec2.EC2, instanceID, failState string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIds: []*string{aws.String(instanceID)},
		})
		if err != nil {
			if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidInstanceID.NotFound" {
				// Set this to nil as if we didn't find anything.
				resp = nil
			} else {
				log.Printf("Error on InstanceStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil || len(resp.Reservations) == 0 || len(resp.Reservations[0].Instances) == 0 {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		i := resp.Reservations[0].Instances[0]
		state := *i.State.Name

		if state == failState {
			return i, state, fmt.Errorf("Failed to reach target state. Reason: %s",
				stringifyStateReason(i.StateReason))

		}

		return i, state, nil
	}
}

func stringifyStateReason(sr *ec2.StateReason) string {
	if sr.Message != nil {
		return *sr.Message
	}
	if sr.Code != nil {
		return *sr.Code
	}

	return sr.String()
}

func readBlockDevices(d *schema.ResourceData, instance *ec2.Instance, conn *ec2.EC2) error {
	ibds, err := readBlockDevicesFromInstance(instance, conn)
	if err != nil {
		return err
	}

	if err := d.Set("ebs_block_device", ibds["ebs"]); err != nil {
		return err
	}

	// This handles the import case which needs to be defaulted to empty
	if _, ok := d.GetOk("root_block_device"); !ok {
		if err := d.Set("root_block_device", []interface{}{}); err != nil {
			return err
		}
	}

	if ibds["root"] != nil {
		roots := []interface{}{ibds["root"]}
		if err := d.Set("root_block_device", roots); err != nil {
			return err
		}
	}

	return nil
}

func readBlockDevicesFromInstance(instance *ec2.Instance, conn *ec2.EC2) (map[string]interface{}, error) {
	blockDevices := make(map[string]interface{})
	blockDevices["ebs"] = make([]map[string]interface{}, 0)
	blockDevices["root"] = nil

	instanceBlockDevices := make(map[string]*ec2.InstanceBlockDeviceMapping)
	for _, bd := range instance.BlockDeviceMappings {
		if bd.Ebs != nil {
			instanceBlockDevices[*bd.Ebs.VolumeId] = bd
		}
	}

	if len(instanceBlockDevices) == 0 {
		return nil, nil
	}

	volIDs := make([]*string, 0, len(instanceBlockDevices))
	for volID := range instanceBlockDevices {
		volIDs = append(volIDs, aws.String(volID))
	}

	// Need to call DescribeVolumes to get volume_size and volume_type for each
	// EBS block device
	volResp, err := conn.DescribeVolumes(&ec2.DescribeVolumesInput{
		VolumeIds: volIDs,
	})
	if err != nil {
		return nil, err
	}

	for _, vol := range volResp.Volumes {
		instanceBd := instanceBlockDevices[*vol.VolumeId]
		bd := make(map[string]interface{})

		bd["volume_id"] = *vol.VolumeId

		if instanceBd.Ebs != nil && instanceBd.Ebs.DeleteOnTermination != nil {
			bd["delete_on_termination"] = *instanceBd.Ebs.DeleteOnTermination
		}
		if vol.Size != nil {
			bd["volume_size"] = *vol.Size
		}
		if vol.VolumeType != nil {
			bd["volume_type"] = *vol.VolumeType
		}
		if vol.Iops != nil {
			bd["iops"] = *vol.Iops
		}

		if blockDeviceIsRoot(instanceBd, instance) {
			blockDevices["root"] = bd
		} else {
			if instanceBd.DeviceName != nil {
				bd["device_name"] = *instanceBd.DeviceName
			}
			if vol.Encrypted != nil {
				bd["encrypted"] = *vol.Encrypted
			}
			if vol.SnapshotId != nil {
				bd["snapshot_id"] = *vol.SnapshotId
			}

			blockDevices["ebs"] = append(blockDevices["ebs"].([]map[string]interface{}), bd)
		}
	}

	return blockDevices, nil
}

func blockDeviceIsRoot(bd *ec2.InstanceBlockDeviceMapping, instance *ec2.Instance) bool {
	return bd.DeviceName != nil &&
		instance.RootDeviceName != nil &&
		*bd.DeviceName == *instance.RootDeviceName
}

func fetchRootDeviceName(ami string, conn *ec2.EC2) (*string, error) {
	if ami == "" {
		return nil, errors.New("Cannot fetch root device name for blank AMI ID.")
	}

	log.Printf("[DEBUG] Describing AMI %q to get root block device name", ami)
	res, err := conn.DescribeImages(&ec2.DescribeImagesInput{
		ImageIds: []*string{aws.String(ami)},
	})
	if err != nil {
		return nil, err
	}

	// For a bad image, we just return nil so we don't block a refresh
	if len(res.Images) == 0 {
		return nil, nil
	}

	image := res.Images[0]
	rootDeviceName := image.RootDeviceName

	// Instance store backed AMIs do not provide a root device name.
	if *image.RootDeviceType == ec2.DeviceTypeInstanceStore {
		return nil, nil
	}

	// Some AMIs have a RootDeviceName like "/dev/sda1" that does not appear as a
	// DeviceName in the BlockDeviceMapping list (which will instead have
	// something like "/dev/sda")
	//
	// While this seems like it breaks an invariant of AMIs, it ends up working
	// on the AWS side, and AMIs like this are common enough that we need to
	// special case it so Terraform does the right thing.
	//
	// Our heuristic is: if the RootDeviceName does not appear in the
	// BlockDeviceMapping, assume that the DeviceName of the first
	// BlockDeviceMapping entry serves as the root device.
	rootDeviceNameInMapping := false
	for _, bdm := range image.BlockDeviceMappings {
		if bdm.DeviceName == image.RootDeviceName {
			rootDeviceNameInMapping = true
		}
	}

	if !rootDeviceNameInMapping && len(image.BlockDeviceMappings) > 0 {
		rootDeviceName = image.BlockDeviceMappings[0].DeviceName
	}

	if rootDeviceName == nil {
		return nil, fmt.Errorf("[WARN] Error finding Root Device Name for AMI (%s)", ami)
	}

	return rootDeviceName, nil
}

func buildNetworkInterfaceOpts(d *schema.ResourceData, groups []*string, nInterfaces interface{}) []*ec2.InstanceNetworkInterfaceSpecification {
	networkInterfaces := []*ec2.InstanceNetworkInterfaceSpecification{}
	// Get necessary items
	subnet, hasSubnet := d.GetOk("subnet_id")

	if hasSubnet {
		// If we have a non-default VPC / Subnet specified, we can flag
		// AssociatePublicIpAddress to get a Public IP assigned. By default these are not provided.
		// You cannot specify both SubnetId and the NetworkInterface.0.* parameters though, otherwise
		// you get: Network interfaces and an instance-level subnet ID may not be specified on the same request
		// You also need to attach Security Groups to the NetworkInterface instead of the instance,
		// to avoid: Network interfaces and an instance-level security groups may not be specified on
		// the same request
		ni := &ec2.InstanceNetworkInterfaceSpecification{
			DeviceIndex: aws.Int64(int64(0)),
			SubnetId:    aws.String(subnet.(string)),
			Groups:      groups,
		}

		if v, ok := d.GetOkExists("associate_public_ip_address"); ok {
			ni.AssociatePublicIpAddress = aws.Bool(v.(bool))
		}

		if v, ok := d.GetOk("private_ip"); ok {
			ni.PrivateIpAddress = aws.String(v.(string))
		}

		if v, ok := d.GetOk("ipv6_address_count"); ok {
			ni.Ipv6AddressCount = aws.Int64(int64(v.(int)))
		}

		if v, ok := d.GetOk("ipv6_addresses"); ok {
			ipv6Addresses := make([]*ec2.InstanceIpv6Address, len(v.([]interface{})))
			for _, address := range v.([]interface{}) {
				ipv6Address := &ec2.InstanceIpv6Address{
					Ipv6Address: aws.String(address.(string)),
				}

				ipv6Addresses = append(ipv6Addresses, ipv6Address)
			}

			ni.Ipv6Addresses = ipv6Addresses
		}

		if v := d.Get("vpc_security_group_ids").(*schema.Set); v.Len() > 0 {
			for _, v := range v.List() {
				ni.Groups = append(ni.Groups, aws.String(v.(string)))
			}
		}

		networkInterfaces = append(networkInterfaces, ni)
	} else {
		// If we have manually specified network interfaces, build and attach those here.
		vL := nInterfaces.(*schema.Set).List()
		for _, v := range vL {
			ini := v.(map[string]interface{})
			ni := &ec2.InstanceNetworkInterfaceSpecification{
				DeviceIndex:         aws.Int64(int64(ini["device_index"].(int))),
				NetworkInterfaceId:  aws.String(ini["network_interface_id"].(string)),
				DeleteOnTermination: aws.Bool(ini["delete_on_termination"].(bool)),
			}
			networkInterfaces = append(networkInterfaces, ni)
		}
	}

	return networkInterfaces
}

func readBlockDeviceMappingsFromConfig(
	d *schema.ResourceData, conn *ec2.EC2) ([]*ec2.BlockDeviceMapping, error) {
	blockDevices := make([]*ec2.BlockDeviceMapping, 0)

	if v, ok := d.GetOk("ebs_block_device"); ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			ebs := &ec2.EbsBlockDevice{
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
				if ec2.VolumeTypeIo1 == strings.ToLower(v) {
					// Condition: This parameter is required for requests to create io1
					// volumes; it is not used in requests to create gp2, st1, sc1, or
					// standard volumes.
					// See: http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_EbsBlockDevice.html
					if v, ok := bd["iops"].(int); ok && v > 0 {
						ebs.Iops = aws.Int64(int64(v))
					}
				}
			}

			blockDevices = append(blockDevices, &ec2.BlockDeviceMapping{
				DeviceName: aws.String(bd["device_name"].(string)),
				Ebs:        ebs,
			})
		}
	}

	if v, ok := d.GetOk("ephemeral_block_device"); ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			bdm := &ec2.BlockDeviceMapping{
				DeviceName:  aws.String(bd["device_name"].(string)),
				VirtualName: aws.String(bd["virtual_name"].(string)),
			}
			if v, ok := bd["no_device"].(bool); ok && v {
				bdm.NoDevice = aws.String("")
				// When NoDevice is true, just ignore VirtualName since it's not needed
				bdm.VirtualName = nil
			}

			if bdm.NoDevice == nil && aws.StringValue(bdm.VirtualName) == "" {
				return nil, errors.New("virtual_name cannot be empty when no_device is false or undefined.")
			}

			blockDevices = append(blockDevices, bdm)
		}
	}

	if v, ok := d.GetOk("root_block_device"); ok {
		vL := v.([]interface{})
		if len(vL) > 1 {
			return nil, errors.New("Cannot specify more than one root_block_device.")
		}
		for _, v := range vL {
			bd := v.(map[string]interface{})
			ebs := &ec2.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(bd["delete_on_termination"].(bool)),
			}

			if v, ok := bd["volume_size"].(int); ok && v != 0 {
				ebs.VolumeSize = aws.Int64(int64(v))
			}

			if v, ok := bd["volume_type"].(string); ok && v != "" {
				ebs.VolumeType = aws.String(v)
			}

			if v, ok := bd["iops"].(int); ok && v > 0 && *ebs.VolumeType == "io1" {
				// Only set the iops attribute if the volume type is io1. Setting otherwise
				// can trigger a refresh/plan loop based on the computed value that is given
				// from AWS, and prevent us from specifying 0 as a valid iops.
				//   See https://github.com/hashicorp/terraform/pull/4146
				//   See https://github.com/hashicorp/terraform/issues/7765
				ebs.Iops = aws.Int64(int64(v))
			} else if v, ok := bd["iops"].(int); ok && v > 0 && *ebs.VolumeType != "io1" {
				// Message user about incompatibility
				log.Print("[WARN] IOPs is only valid for storate type io1 for EBS Volumes")
			}

			if dn, err := fetchRootDeviceName(d.Get("ami").(string), conn); err == nil {
				if dn == nil {
					return nil, fmt.Errorf(
						"Expected 1 AMI for ID: %s, got none",
						d.Get("ami").(string))
				}

				blockDevices = append(blockDevices, &ec2.BlockDeviceMapping{
					DeviceName: dn,
					Ebs:        ebs,
				})
			} else {
				return nil, err
			}
		}
	}

	return blockDevices, nil
}

func readVolumeTags(conn *ec2.EC2, d *schema.ResourceData) error {
	volumeIds, err := getAwsInstanceVolumeIds(conn, d)
	if err != nil {
		return err
	}

	tagsResp, err := conn.DescribeTags(&ec2.DescribeTagsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("resource-id"),
				Values: volumeIds,
			},
		},
	})
	if err != nil {
		return err
	}

	var tags []*ec2.Tag

	for _, t := range tagsResp.Tags {
		tag := &ec2.Tag{
			Key:   t.Key,
			Value: t.Value,
		}
		tags = append(tags, tag)
	}

	d.Set("volume_tags", tagsToMap(tags))

	return nil
}

// Determine whether we're referring to security groups with
// IDs or names. We use a heuristic to figure this out. By default,
// we use IDs if we're in a VPC, and names otherwise (EC2-Classic).
// However, the default VPC accepts either, so store them both here and let the
// config determine which one to use in Plan and Apply.
func readSecurityGroups(d *schema.ResourceData, instance *ec2.Instance, conn *ec2.EC2) error {
	// An instance with a subnet is in a VPC; an instance without a subnet is in EC2-Classic.
	hasSubnet := instance.SubnetId != nil && *instance.SubnetId != ""
	useID, useName := hasSubnet, !hasSubnet

	// If the instance is in a VPC, find out if that VPC is Default to determine
	// whether to store names.
	if instance.VpcId != nil && *instance.VpcId != "" {
		out, err := conn.DescribeVpcs(&ec2.DescribeVpcsInput{
			VpcIds: []*string{instance.VpcId},
		})
		if err != nil {
			log.Printf("[WARN] Unable to describe VPC %q: %s", *instance.VpcId, err)
		} else if len(out.Vpcs) == 0 {
			// This may happen in Eucalyptus Cloud
			log.Printf("[WARN] Unable to retrieve VPCs")
		} else {
			isInDefaultVpc := *out.Vpcs[0].IsDefault
			useName = isInDefaultVpc
		}
	}

	// Build up the security groups
	if useID {
		sgs := make([]string, 0, len(instance.SecurityGroups))
		for _, sg := range instance.SecurityGroups {
			sgs = append(sgs, *sg.GroupId)
		}
		log.Printf("[DEBUG] Setting Security Group IDs: %#v", sgs)
		if err := d.Set("vpc_security_group_ids", sgs); err != nil {
			return err
		}
	} else {
		if err := d.Set("vpc_security_group_ids", []string{}); err != nil {
			return err
		}
	}
	if useName {
		sgs := make([]string, 0, len(instance.SecurityGroups))
		for _, sg := range instance.SecurityGroups {
			sgs = append(sgs, *sg.GroupName)
		}
		log.Printf("[DEBUG] Setting Security Group Names: %#v", sgs)
		if err := d.Set("security_groups", sgs); err != nil {
			return err
		}
	} else {
		if err := d.Set("security_groups", []string{}); err != nil {
			return err
		}
	}
	return nil
}

type awsInstanceOpts struct {
	BlockDeviceMappings               []*ec2.BlockDeviceMapping
	DisableAPITermination             *bool
	EBSOptimized                      *bool
	Monitoring                        *ec2.RunInstancesMonitoringEnabled
	IAMInstanceProfile                *ec2.IamInstanceProfileSpecification
	ImageID                           *string
	InstanceInitiatedShutdownBehavior *string
	InstanceType                      *string
	Ipv6AddressCount                  *int64
	Ipv6Addresses                     []*ec2.InstanceIpv6Address
	KeyName                           *string
	NetworkInterfaces                 []*ec2.InstanceNetworkInterfaceSpecification
	Placement                         *ec2.Placement
	PrivateIPAddress                  *string
	SecurityGroupIDs                  []*string
	SecurityGroups                    []*string
	SpotPlacement                     *ec2.SpotPlacement
	SubnetID                          *string
	UserData64                        *string
}

func buildAwsInstanceOpts(
	d *schema.ResourceData, meta interface{}) (*awsInstanceOpts, error) {
	conn := meta.(*AWSClient).ec2conn

	opts := &awsInstanceOpts{
		DisableAPITermination: aws.Bool(d.Get("disable_api_termination").(bool)),
		EBSOptimized:          aws.Bool(d.Get("ebs_optimized").(bool)),
		ImageID:               aws.String(d.Get("ami").(string)),
		InstanceType:          aws.String(d.Get("instance_type").(string)),
	}

	if v := d.Get("instance_initiated_shutdown_behavior").(string); v != "" {
		opts.InstanceInitiatedShutdownBehavior = aws.String(v)
	}

	opts.Monitoring = &ec2.RunInstancesMonitoringEnabled{
		Enabled: aws.Bool(d.Get("monitoring").(bool)),
	}

	opts.IAMInstanceProfile = &ec2.IamInstanceProfileSpecification{
		Name: aws.String(d.Get("iam_instance_profile").(string)),
	}

	userData := d.Get("user_data").(string)
	userDataBase64 := d.Get("user_data_base64").(string)

	if userData != "" {
		opts.UserData64 = aws.String(base64Encode([]byte(userData)))
	} else if userDataBase64 != "" {
		opts.UserData64 = aws.String(userDataBase64)
	}

	// check for non-default Subnet, and cast it to a String
	subnet, hasSubnet := d.GetOk("subnet_id")
	subnetID := subnet.(string)

	// Placement is used for aws_instance; SpotPlacement is used for
	// aws_spot_instance_request. They represent the same data. :-|
	opts.Placement = &ec2.Placement{
		AvailabilityZone: aws.String(d.Get("availability_zone").(string)),
		GroupName:        aws.String(d.Get("placement_group").(string)),
	}

	opts.SpotPlacement = &ec2.SpotPlacement{
		AvailabilityZone: aws.String(d.Get("availability_zone").(string)),
		GroupName:        aws.String(d.Get("placement_group").(string)),
	}

	if v := d.Get("tenancy").(string); v != "" {
		opts.Placement.Tenancy = aws.String(v)
	}

	var groups []*string
	if v := d.Get("security_groups"); v != nil {
		// Security group names.
		// For a nondefault VPC, you must use security group IDs instead.
		// See http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_RunInstances.html
		sgs := v.(*schema.Set).List()
		if len(sgs) > 0 && hasSubnet {
			log.Print("[WARN] Deprecated. Attempting to use 'security_groups' within a VPC instance. Use 'vpc_security_group_ids' instead.")
		}
		for _, v := range sgs {
			str := v.(string)
			groups = append(groups, aws.String(str))
		}
	}

	networkInterfaces, interfacesOk := d.GetOk("network_interface")

	// If setting subnet and public address, OR manual network interfaces, populate those now.
	if hasSubnet || interfacesOk {
		// Otherwise we're attaching (a) network interface(s)
		opts.NetworkInterfaces = buildNetworkInterfaceOpts(d, groups, networkInterfaces)
	} else {
		// If simply specifying a subnetID, privateIP, Security Groups, or VPC Security Groups, build these now
		if subnetID != "" {
			opts.SubnetID = aws.String(subnetID)
		}

		if v, ok := d.GetOk("private_ip"); ok {
			opts.PrivateIPAddress = aws.String(v.(string))
		}
		if opts.SubnetID != nil &&
			*opts.SubnetID != "" {
			opts.SecurityGroupIDs = groups
		} else {
			opts.SecurityGroups = groups
		}

		if v, ok := d.GetOk("ipv6_address_count"); ok {
			opts.Ipv6AddressCount = aws.Int64(int64(v.(int)))
		}

		if v, ok := d.GetOk("ipv6_addresses"); ok {
			ipv6Addresses := make([]*ec2.InstanceIpv6Address, len(v.([]interface{})))
			for _, address := range v.([]interface{}) {
				ipv6Address := &ec2.InstanceIpv6Address{
					Ipv6Address: aws.String(address.(string)),
				}

				ipv6Addresses = append(ipv6Addresses, ipv6Address)
			}

			opts.Ipv6Addresses = ipv6Addresses
		}

		if v := d.Get("vpc_security_group_ids").(*schema.Set); v.Len() > 0 {
			for _, v := range v.List() {
				opts.SecurityGroupIDs = append(opts.SecurityGroupIDs, aws.String(v.(string)))
			}
		}
	}

	if v, ok := d.GetOk("key_name"); ok {
		opts.KeyName = aws.String(v.(string))
	}

	blockDevices, err := readBlockDeviceMappingsFromConfig(d, conn)
	if err != nil {
		return nil, err
	}
	if len(blockDevices) > 0 {
		opts.BlockDeviceMappings = blockDevices
	}
	return opts, nil
}

func awsTerminateInstance(conn *ec2.EC2, id string, d *schema.ResourceData) error {
	log.Printf("[INFO] Terminating instance: %s", id)
	req := &ec2.TerminateInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	}
	if _, err := conn.TerminateInstances(req); err != nil {
		return fmt.Errorf("Error terminating instance: %s", err)
	}

	log.Printf("[DEBUG] Waiting for instance (%s) to become terminated", id)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending", "running", "shutting-down", "stopped", "stopping"},
		Target:     []string{"terminated"},
		Refresh:    InstanceStateRefreshFunc(conn, id, ""),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for instance (%s) to terminate: %s", id, err)
	}

	return nil
}

func iamInstanceProfileArnToName(ip *ec2.IamInstanceProfile) string {
	if ip == nil || ip.Arn == nil {
		return ""
	}
	parts := strings.Split(*ip.Arn, "/")
	return parts[len(parts)-1]
}

func userDataHashSum(user_data string) string {
	// Check whether the user_data is not Base64 encoded.
	// Always calculate hash of base64 decoded value since we
	// check against double-encoding when setting it
	v, base64DecodeError := base64.StdEncoding.DecodeString(user_data)
	if base64DecodeError != nil {
		v = []byte(user_data)
	}

	hash := sha1.Sum(v)
	return hex.EncodeToString(hash[:])
}

func getAwsInstanceVolumeIds(conn *ec2.EC2, d *schema.ResourceData) ([]*string, error) {
	volumeIds := make([]*string, 0)

	opts := &ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("attachment.instance-id"),
				Values: []*string{aws.String(d.Id())},
			},
		},
	}

	resp, err := conn.DescribeVolumes(opts)
	if err != nil {
		return nil, err
	}

	for _, v := range resp.Volumes {
		volumeIds = append(volumeIds, v.VolumeId)
	}

	return volumeIds, nil
}
