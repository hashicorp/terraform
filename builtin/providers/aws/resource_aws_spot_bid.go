package aws

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	//	"strings"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	//	"github.com/awslabs/aws-sdk-go/aws/awserr"
	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSpotBid() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSpotBidCreate,
		Read:   resourceAwsSpotBidRead,
		Update: resourceAwsSpotBidUpdate,
		Delete: resourceAwsSpotBidDelete,

		SchemaVersion: 1,
		//		MigrateState:  resourceAwsSpotBidMigrateState, // FIXME: not implemented

		Schema: map[string]*schema.Schema{
			"ami": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"associate_public_ip_address": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"placement_group": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
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

			"spot_persist": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},

			"spot_price": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"source_dest_check": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
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
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"vpc_security_group_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},

			"public_dns": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"public_ip": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"private_dns": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
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

			"tenancy": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"tags": tagsSchema(),

			"block_device": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				Removed:  "Split out into three sub-types; see Changelog and Docs",
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
		},
	}
}

func resourceAwsSpotBidCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	// Figure out user data
	userData := ""
	if v := d.Get("user_data"); v != nil {
		userData = base64.StdEncoding.EncodeToString([]byte(v.(string)))
	}

	// check for non-default Subnet, and cast it to a String
	var hasSubnet bool
	subnet, hasSubnet := d.GetOk("subnet_id")
	subnetID := subnet.(string)

	placement := &ec2.SpotPlacement{
		AvailabilityZone: aws.String(d.Get("availability_zone").(string)),
		GroupName:        aws.String(d.Get("placement_group").(string)),
	}

	iam := &ec2.IAMInstanceProfileSpecification{
		Name: aws.String(d.Get("iam_instance_profile").(string)),
	}

	// Build the creation struct
	launchSpec := &ec2.RequestSpotLaunchSpecification{
		ImageID:            aws.String(d.Get("ami").(string)),
		Placement:          placement,
		InstanceType:       aws.String(d.Get("instance_type").(string)),
		UserData:           aws.String(userData),
		EBSOptimized:       aws.Boolean(d.Get("ebs_optimized").(bool)),
		IAMInstanceProfile: iam,
	}

	associatePublicIPAddress := false
	if v := d.Get("associate_public_ip_address"); v != nil {
		associatePublicIPAddress = v.(bool)
	}

	var groups []*string
	if v := d.Get("security_groups"); v != nil {
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
			AssociatePublicIPAddress: aws.Boolean(associatePublicIPAddress),
			DeviceIndex:              aws.Long(int64(0)),
			SubnetID:                 aws.String(subnetID),
			Groups:                   groups,
		}

		if v, ok := d.GetOk("private_ip"); ok {
			ni.PrivateIPAddress = aws.String(v.(string))
		}

		if v := d.Get("vpc_security_group_ids"); v != nil {
			for _, v := range v.(*schema.Set).List() {
				ni.Groups = append(ni.Groups, aws.String(v.(string)))
			}
		}

		launchSpec.NetworkInterfaces = []*ec2.InstanceNetworkInterfaceSpecification{ni}
	} else {
		if subnetID != "" {
			launchSpec.SubnetID = aws.String(subnetID)
		}

		if launchSpec.SubnetID != nil &&
			*launchSpec.SubnetID != "" {
			launchSpec.SecurityGroupIDs = groups
		} else {
			launchSpec.SecurityGroups = groups
		}

		if v := d.Get("vpc_security_group_ids"); v != nil {
			for _, v := range v.(*schema.Set).List() {
				launchSpec.SecurityGroupIDs = append(launchSpec.SecurityGroupIDs, aws.String(v.(string)))
			}
		}
	}

	if v, ok := d.GetOk("key_name"); ok {
		launchSpec.KeyName = aws.String(v.(string))
	}

	blockDevices := make([]*ec2.BlockDeviceMapping, 0)

	if v, ok := d.GetOk("ebs_block_device"); ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			ebs := &ec2.EBSBlockDevice{
				DeleteOnTermination: aws.Boolean(bd["delete_on_termination"].(bool)),
				Encrypted:           aws.Boolean(bd["encrypted"].(bool)),
			}

			if v, ok := bd["snapshot_id"].(string); ok && v != "" {
				ebs.SnapshotID = aws.String(v)
			}

			if v, ok := bd["volume_size"].(int); ok && v != 0 {
				ebs.VolumeSize = aws.Long(int64(v))
			}

			if v, ok := bd["volume_type"].(string); ok && v != "" {
				ebs.VolumeType = aws.String(v)
			}

			if v, ok := bd["iops"].(int); ok && v > 0 {
				ebs.IOPS = aws.Long(int64(v))
			}

			blockDevices = append(blockDevices, &ec2.BlockDeviceMapping{
				DeviceName: aws.String(bd["device_name"].(string)),
				EBS:        ebs,
			})
		}
	}

	if v, ok := d.GetOk("ephemeral_block_device"); ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			blockDevices = append(blockDevices, &ec2.BlockDeviceMapping{
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
			ebs := &ec2.EBSBlockDevice{
				DeleteOnTermination: aws.Boolean(bd["delete_on_termination"].(bool)),
			}

			if v, ok := bd["volume_size"].(int); ok && v != 0 {
				ebs.VolumeSize = aws.Long(int64(v))
			}

			if v, ok := bd["volume_type"].(string); ok && v != "" {
				ebs.VolumeType = aws.String(v)
			}

			if v, ok := bd["iops"].(int); ok && v > 0 {
				ebs.IOPS = aws.Long(int64(v))
			}

			if dn, err := fetchRootDeviceName(d.Get("ami").(string), conn); err == nil {
				if dn == nil {
					return fmt.Errorf(
						"Expected 1 AMI for ID: %s, got none",
						d.Get("ami").(string))
				}

				blockDevices = append(blockDevices, &ec2.BlockDeviceMapping{
					DeviceName: dn,
					EBS:        ebs,
				})
			} else {
				return err
			}
		}
	}

	if len(blockDevices) > 0 {
		launchSpec.BlockDeviceMappings = blockDevices
	}

	spotType := "one-time"

	if d.Get("spot_persist").(bool) {
		spotType = "persistent"
	}

	spotOpts := &ec2.RequestSpotInstancesInput{
		SpotPrice:           aws.String(d.Get("spot_price").(string)),
		Type:                aws.String(spotType),
		InstanceCount:       aws.Long(1),
		LaunchSpecification: launchSpec,
	}

	// Make the spot instance request
	var spotResp *ec2.RequestSpotInstancesOutput
	spotResp, err := conn.RequestSpotInstances(spotOpts)

	request_id := *spotResp.SpotInstanceRequests[0].SpotInstanceRequestID
	d.SetId(request_id)

	spotStateConf := &resource.StateChangeConf{
		// http://docs.aws.amazon.com/AWSEC2/latest/UserGuide/spot-bid-status.html
		Pending:    []string{"start", "pending-evaluation", "pending-fulfillment"},
		Target:     "fulfilled",
		Refresh:    SpotInstanceStateRefreshFunc(conn, request_id),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	log.Printf("[DEBUG] waiting for spot bid to resolve... this may take several minutes.")
	_, err = spotStateConf.WaitForState()

	if err != nil {
		return fmt.Errorf("Error while waiting for spot request (%s) to resolve: %s", request_id, err)
	}

	return nil
}

// Update spot state, etc
func resourceAwsSpotBidRead(d *schema.ResourceData, meta interface{}) error {
	//conn := meta.(*AWSClient).ec2conn

	// FIXME implement to actually update state

	return nil
}

// Bids are not mutable, except maybe tags?
func resourceAwsSpotBidUpdate(d *schema.ResourceData, meta interface{}) error {
	//conn := meta.(*AWSClient).ec2conn

	d.Partial(true)

	// FIXME either do something here or decide explicitly to do nothing

	d.Partial(false)

	return resourceAwsInstanceRead(d, meta)
}

func resourceAwsSpotBidDelete(d *schema.ResourceData, meta interface{}) error {
	//conn := meta.(*AWSClient).ec2conn

	// FIXME actually delete this

	d.SetId("")
	return nil
}

// SpotInstanceStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an EC2 spot instance request
func SpotInstanceStateRefreshFunc(conn *ec2.EC2, reqID string) resource.StateRefreshFunc {

	//SpotInstanceRequests

	return func() (interface{}, string, error) {
		resp, err := conn.DescribeSpotInstanceRequests(&ec2.DescribeSpotInstanceRequestsInput{
			SpotInstanceRequestIDs: []*string{aws.String(reqID)},
		})

		// FIXME: actually do error handling instead of happy path
		if err != nil {
			/*
			   if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "InvalidInstanceID.NotFound" {
			     // Set this to nil as if we didn't find anything.
			     resp = nil
			   } else {
			     log.Printf("Error on InstanceStateRefresh: %s", err)
			     return nil, "", err
			   }
			*/
		}

		if resp == nil || len(resp.SpotInstanceRequests) == 0 {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		req := resp.SpotInstanceRequests[0]
		return req, *req.Status.Code, nil
	}
}
