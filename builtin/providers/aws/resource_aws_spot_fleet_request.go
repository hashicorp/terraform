package aws

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSpotFleetRequest() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSpotFleetRequestCreate,
		Read:   resourceAwsSpotFleetRequestRead,
		Delete: resourceAwsSpotFleetRequestDelete,
		Update: resourceAwsSpotFleetRequestUpdate,

		Schema: map[string]*schema.Schema{
			"iam_fleet_role": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			// http://docs.aws.amazon.com/sdk-for-go/api/service/ec2.html#type-SpotFleetLaunchSpecification
			// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_SpotFleetLaunchSpecification.html
			"launch_specification": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
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
									"encrypted": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
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
							Computed: true,
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
						"ebs_optimized": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						"iam_instance_profile": &schema.Schema{
							Type:     schema.TypeString,
							ForceNew: true,
							Optional: true,
						},
						"ami": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"instance_type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"key_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Computed: true,
						},
						"monitoring": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
						},
						//									"network_interface_set"
						"placement_group": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
						"spot_price": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"subnet_id": &schema.Schema{
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
						"weighted_capacity": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"availability_zone": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-", m["availability_zone"].(string)))
					buf.WriteString(fmt.Sprintf("%s-", m["instance_type"].(string)))
					return hashcode.String(buf.String())
				},
			},
			// Everything on a spot fleet is ForceNew except target_capacity
			"target_capacity": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: false,
			},
			"allocation_strategy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"excess_capacity_termination_policy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"spot_price": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"terminate_instances_with_expiration": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
			"valid_from": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"valid_until": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func buildSpotFleetLaunchSpecification(d map[string]interface{}, meta interface{}) (*ec2.SpotFleetLaunchSpecification, error) {
	conn := meta.(*AWSClient).ec2conn

	opts := &ec2.SpotFleetLaunchSpecification{
		ImageId:      aws.String(d["ami"].(string)),
		InstanceType: aws.String(d["instance_type"].(string)),
		SpotPrice:    aws.String(d["spot_price"].(string)),
	}

	if v, ok := d["ebs_optimized"]; ok {
		opts.EbsOptimized = aws.Bool(v.(bool))
	}

	if v, ok := d["monitoring"]; ok {
		opts.Monitoring = &ec2.SpotFleetMonitoring{
			Enabled: aws.Bool(v.(bool)),
		}
	}

	if v, ok := d["iam_instance_profile"]; ok {
		opts.IamInstanceProfile = &ec2.IamInstanceProfileSpecification{
			Name: aws.String(v.(string)),
		}
	}

	if v, ok := d["user_data"]; ok {
		opts.UserData = aws.String(
			base64.StdEncoding.EncodeToString([]byte(v.(string))))
	}

	// check for non-default Subnet, and cast it to a String
	subnet, hasSubnet := d["subnet_id"]
	subnetID := subnet.(string)

	var associatePublicIPAddress bool
	if v, ok := d["associate_public_ip_address"]; ok {
		associatePublicIPAddress = v.(bool)
	}

	var groups []*string
	if v, ok := d["security_groups"]; ok {
		// Security group names.
		// For a nondefault VPC, you must use security group IDs instead.
		// See http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_RunInstances.html
		sgs := v.(*schema.Set).List()
		if len(sgs) > 0 && hasSubnet {
			log.Printf("[WARN] Deprecated. Attempting to use 'security_groups' within a VPC instance. Use 'vpc_security_group_ids' instead.")
		}
		for _, v := range sgs {
			str := v.(string)
			groups = append(groups, aws.String(str))
		}
	}

	if hasSubnet && associatePublicIPAddress {
		// If we have a non-default VPC / Subnet specified, we can flag
		// AssociatePublicIpAddress to get a Public IP assigned. By default these are not provided.
		// You cannot specify both SubnetId and the NetworkInterface.0.* parameters though, otherwise
		// you get: Network interfaces and an instance-level subnet ID may not be specified on the same request
		// You also need to attach Security Groups to the NetworkInterface instead of the instance,
		// to avoid: Network interfaces and an instance-level security groups may not be specified on
		// the same request
		ni := &ec2.InstanceNetworkInterfaceSpecification{
			AssociatePublicIpAddress: aws.Bool(associatePublicIPAddress),
			DeviceIndex:              aws.Int64(int64(0)),
			SubnetId:                 aws.String(subnetID),
			Groups:                   groups,
		}

		if v, ok := d["private_ip"]; ok {
			ni.PrivateIpAddress = aws.String(v.(string))
		}

		if v := d["vpc_security_group_ids"].(*schema.Set); v.Len() > 0 {
			for _, v := range v.List() {
				ni.Groups = append(ni.Groups, aws.String(v.(string)))
			}
		}

		opts.NetworkInterfaces = []*ec2.InstanceNetworkInterfaceSpecification{ni}
	} else {
		if subnetID != "" {
			opts.SubnetId = aws.String(subnetID)
		}

		if v, ok := d["vpc_security_group_ids"]; ok {
			if s := v.(*schema.Set); s.Len() > 0 {
				for _, v := range s.List() {
					opts.SecurityGroups = append(opts.SecurityGroups, &ec2.GroupIdentifier{GroupId: aws.String(v.(string))})
				}
			}
		}
	}

	if v, ok := d["key_name"]; ok {
		opts.KeyName = aws.String(v.(string))
	}

	blockDevices, err := readSpotFleetBlockDeviceMappingsFromConfig(d, conn)
	if err != nil {
		return nil, err
	}
	if len(blockDevices) > 0 {
		opts.BlockDeviceMappings = blockDevices
	}

	return opts, nil
}

func readSpotFleetBlockDeviceMappingsFromConfig(
	d map[string]interface{}, conn *ec2.EC2) ([]*ec2.BlockDeviceMapping, error) {
	blockDevices := make([]*ec2.BlockDeviceMapping, 0)

	if v, ok := d["ebs_block_device"]; ok {
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
			}

			if v, ok := bd["iops"].(int); ok && v > 0 {
				ebs.Iops = aws.Int64(int64(v))
			}

			blockDevices = append(blockDevices, &ec2.BlockDeviceMapping{
				DeviceName: aws.String(bd["device_name"].(string)),
				Ebs:        ebs,
			})
		}
	}

	if v, ok := d["ephemeral_block_device"]; ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			blockDevices = append(blockDevices, &ec2.BlockDeviceMapping{
				DeviceName:  aws.String(bd["device_name"].(string)),
				VirtualName: aws.String(bd["virtual_name"].(string)),
			})
		}
	}

	if v, ok := d["root_block_device"]; ok {
		vL := v.(*schema.Set).List()
		if len(vL) > 1 {
			return nil, fmt.Errorf("Cannot specify more than one root_block_device.")
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

			if v, ok := bd["iops"].(int); ok && v > 0 {
				ebs.Iops = aws.Int64(int64(v))
			}

			if dn, err := fetchRootDeviceName(d["ami"].(string), conn); err == nil {
				if dn == nil {
					return nil, fmt.Errorf(
						"Expected 1 AMI for ID: %s, got none",
						d["ami"].(string))
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

func buildAwsSpotFleetLaunchSpecifications(
	d *schema.ResourceData, meta interface{}) ([]*ec2.SpotFleetLaunchSpecification, error) {
	specs := []*ec2.SpotFleetLaunchSpecification{}
	user_specs := d.Get("launch_specification").(*schema.Set).List()
	for _, user_spec := range user_specs {
		user_spec_map := user_spec.(map[string]interface{})
		// panic: interface conversion: interface {} is map[string]interface {}, not *schema.ResourceData
		opts, err := buildSpotFleetLaunchSpecification(user_spec_map, meta)
		if err != nil {
			return nil, err
		}
		specs = append(specs, opts)
	}

	return specs, nil
}

func resourceAwsSpotFleetRequestCreate(d *schema.ResourceData, meta interface{}) error {
	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_RequestSpotFleet.html
	conn := meta.(*AWSClient).ec2conn

	launch_specs, err := buildAwsSpotFleetLaunchSpecifications(d, meta)
	if err != nil {
		return err
	}

	// http://docs.aws.amazon.com/sdk-for-go/api/service/ec2.html#type-SpotFleetRequestConfigData
	spotFleetConfig := &ec2.SpotFleetRequestConfigData{
		IamFleetRole:                     aws.String(d.Get("iam_fleet_role").(string)),
		LaunchSpecifications:             launch_specs,
		SpotPrice:                        aws.String(d.Get("spot_price").(string)),
		TargetCapacity:                   aws.Int64(int64(d.Get("target_capacity").(int))),
		ClientToken:                      aws.String(resource.UniqueId()),
		TerminateInstancesWithExpiration: aws.Bool(d.Get("terminate_instances_with_expiration").(bool)),
		ValidFrom:                        aws.Time(time.Now()),
	}

	if v, ok := d.GetOk("excess_capacity_termination_policy"); ok {
		spotFleetConfig.ExcessCapacityTerminationPolicy = aws.String(v.(string))
	}

	if v, ok := d.GetOk("allocation_strategy"); ok {
		spotFleetConfig.AllocationStrategy = aws.String(v.(string))
	} else {
		spotFleetConfig.AllocationStrategy = aws.String("lowestPrice")
	}

	if v, ok := d.GetOk("valid_until"); ok {
		valid_until, err := time.Parse(awsAutoscalingScheduleTimeLayout, v.(string))
		if err != nil {
			return err
		}
		spotFleetConfig.ValidUntil = &valid_until
	} else {
		valid_until := time.Now().Add(24 * time.Hour)
		spotFleetConfig.ValidUntil = &valid_until
	}

	// http://docs.aws.amazon.com/sdk-for-go/api/service/ec2.html#type-RequestSpotFleetInput
	spotFleetOpts := &ec2.RequestSpotFleetInput{
		SpotFleetRequestConfig: spotFleetConfig,
		DryRun:                 aws.Bool(false),
	}

	log.Printf("[DEBUG] Requesting spot fleet with these opts: %s", spotFleetOpts)
	resp, err := conn.RequestSpotFleet(spotFleetOpts)
	if err != nil {
		return fmt.Errorf("Error requesting spot fleet: %s", err)
	}

	d.SetId(*resp.SpotFleetRequestId)

	return resourceAwsSpotFleetRequestUpdate(d, meta)
}

func resourceAwsSpotFleetRequestRead(d *schema.ResourceData, meta interface{}) error {
	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeSpotFleetRequests.html
	conn := meta.(*AWSClient).ec2conn

	req := &ec2.DescribeSpotFleetRequestsInput{
		SpotFleetRequestIds: []*string{aws.String(d.Id())},
	}
	resp, err := conn.DescribeSpotFleetRequests(req)

	if err != nil {
		// If the spot request was not found, return nil so that we can show
		// that it is gone.
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidSpotFleetRequestID.NotFound" {
			d.SetId("")
			return nil
		}

		// Some other error, report it
		return err
	}

	request := resp.SpotFleetRequestConfigs[0]

	// if the request is cancelled, then it is gone
	if *request.SpotFleetRequestState == "cancelled" {
		d.SetId("")
		return nil
	}

	d.Set("spot_request_state", request.SpotFleetRequestState)
	return nil
}

func resourceAwsSpotFleetRequestUpdate(d *schema.ResourceData, meta interface{}) error {
	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_ModifySpotFleetRequest.html
	//	conn := meta.(*AWSClient).ec2conn

	d.Partial(true)
	// TODO: Adjust target capacity

	return resourceAwsSpotFleetRequestRead(d, meta)
}

func resourceAwsSpotFleetRequestDelete(d *schema.ResourceData, meta interface{}) error {
	// http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_CancelSpotFleetRequests.html
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[INFO] Cancelling spot fleet request: %s", d.Id())
	_, err := conn.CancelSpotFleetRequests(&ec2.CancelSpotFleetRequestsInput{
		SpotFleetRequestIds: []*string{aws.String(d.Id())},
		TerminateInstances:  aws.Bool(true),
	})

	if err != nil {
		return fmt.Errorf("Error cancelling spot request (%s): %s", d.Id(), err)
	}

	return nil
}
