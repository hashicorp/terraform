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

	"github.com/hashicorp/aws-sdk-go/aws"
	"github.com/hashicorp/aws-sdk-go/gen/ec2"
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

			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"private_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
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

						"iops": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceAwsInstanceBlockDevicesHash,
			},

			"root_block_device": &schema.Schema{
				// TODO: This is a list because we don't support singleton
				//       sub-resources today. We'll enforce that the list only ever has
				//       length zero or one below. When TF gains support for
				//       sub-resources this can be converted.
				Type:     schema.TypeList,
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

						"device_name": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							Default:  "/dev/sda1",
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

						"iops": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	// Figure out user data
	userData := ""
	if v := d.Get("user_data"); v != nil {
		userData = base64.StdEncoding.EncodeToString([]byte(v.(string)))
	}

	placement := &ec2.Placement{
		AvailabilityZone: aws.String(d.Get("availability_zone").(string)),
		Tenancy:          aws.String(d.Get("tenancy").(string)),
	}

	iam := &ec2.IAMInstanceProfileSpecification{
		Name: aws.String(d.Get("iam_instance_profile").(string)),
	}

	// Build the creation struct
	runOpts := &ec2.RunInstancesRequest{
		ImageID:            aws.String(d.Get("ami").(string)),
		Placement:          placement,
		InstanceType:       aws.String(d.Get("instance_type").(string)),
		MaxCount:           aws.Integer(1),
		MinCount:           aws.Integer(1),
		UserData:           aws.String(userData),
		EBSOptimized:       aws.Boolean(d.Get("ebs_optimized").(bool)),
		IAMInstanceProfile: iam,
	}

	associatePublicIPAddress := false
	if v := d.Get("associate_public_ip_address"); v != nil {
		associatePublicIPAddress = v.(bool)
	}

	// check for non-default Subnet, and cast it to a String
	var hasSubnet bool
	subnet, hasSubnet := d.GetOk("subnet_id")
	subnetID := subnet.(string)

	var groups []string
	if v := d.Get("security_groups"); v != nil {
		// Security group names.
		// For a nondefault VPC, you must use security group IDs instead.
		// See http://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_RunInstances.html
		for _, v := range v.(*schema.Set).List() {
			str := v.(string)
			groups = append(groups, str)
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
		ni := ec2.InstanceNetworkInterfaceSpecification{
			AssociatePublicIPAddress: aws.Boolean(associatePublicIPAddress),
			DeviceIndex:              aws.Integer(0),
			SubnetID:                 aws.String(subnetID),
		}

		if v, ok := d.GetOk("private_ip"); ok {
			ni.PrivateIPAddress = aws.String(v.(string))
		}

		if len(groups) > 0 {
			ni.Groups = groups
		}

		runOpts.NetworkInterfaces = []ec2.InstanceNetworkInterfaceSpecification{ni}
	} else {
		if subnetID != "" {
			runOpts.SubnetID = aws.String(subnetID)
		}

		if v, ok := d.GetOk("private_ip"); ok {
			runOpts.PrivateIPAddress = aws.String(v.(string))
		}
		if runOpts.SubnetID != nil &&
			*runOpts.SubnetID != "" {
			runOpts.SecurityGroupIDs = groups
		} else {
			runOpts.SecurityGroups = groups
		}
	}

	if v, ok := d.GetOk("key_name"); ok {
		runOpts.KeyName = aws.String(v.(string))
	}

	blockDevices := make([]interface{}, 0)

	if v := d.Get("block_device"); v != nil {
		blockDevices = append(blockDevices, v.(*schema.Set).List()...)
	}

	if v := d.Get("root_block_device"); v != nil {
		rootBlockDevices := v.([]interface{})
		if len(rootBlockDevices) > 1 {
			return fmt.Errorf("Cannot specify more than one root_block_device.")
		}
		blockDevices = append(blockDevices, rootBlockDevices...)
	}

	if len(blockDevices) > 0 {
		runOpts.BlockDeviceMappings = make([]ec2.BlockDeviceMapping, len(blockDevices))
		for i, v := range blockDevices {
			bd := v.(map[string]interface{})
			runOpts.BlockDeviceMappings[i].DeviceName = aws.String(bd["device_name"].(string))
			runOpts.BlockDeviceMappings[i].EBS = &ec2.EBSBlockDevice{
				VolumeType:          aws.String(bd["volume_type"].(string)),
				VolumeSize:          aws.Integer(bd["volume_size"].(int)),
				DeleteOnTermination: aws.Boolean(bd["delete_on_termination"].(bool)),
			}

			if v, ok := bd["virtual_name"].(string); ok {
				runOpts.BlockDeviceMappings[i].VirtualName = aws.String(v)
			}
			if v, ok := bd["snapshot_id"].(string); ok && v != "" {
				runOpts.BlockDeviceMappings[i].EBS.SnapshotID = aws.String(v)
			}
			if v, ok := bd["encrypted"].(bool); ok {
				runOpts.BlockDeviceMappings[i].EBS.Encrypted = aws.Boolean(v)
			}
			if v, ok := bd["iops"].(int); ok && v > 0 {
				runOpts.BlockDeviceMappings[i].EBS.IOPS = aws.Integer(v)
			}
		}
	}

	// Create the instance
	log.Printf("[DEBUG] Run configuration: %#v", runOpts)
	runResp, err := ec2conn.RunInstances(runOpts)
	if err != nil {
		return fmt.Errorf("Error launching source instance: %s", err)
	}

	instance := &runResp.Instances[0]
	log.Printf("[INFO] Instance ID: %s", *instance.InstanceID)

	// Store the resulting ID so we can look this up later
	d.SetId(*instance.InstanceID)

	// Wait for the instance to become running so we can get some attributes
	// that aren't available until later.
	log.Printf(
		"[DEBUG] Waiting for instance (%s) to become running",
		*instance.InstanceID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     "running",
		Refresh:    InstanceStateRefreshFunc(ec2conn, *instance.InstanceID),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	instanceRaw, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for instance (%s) to become ready: %s",
			*instance.InstanceID, err)
	}

	instance = instanceRaw.(*ec2.Instance)

	// Initialize the connection info
	if instance.PublicIPAddress != nil {
		d.SetConnInfo(map[string]string{
			"type": "ssh",
			"host": *instance.PublicIPAddress,
		})
	}

	// Set our attributes
	if err := resourceAwsInstanceRead(d, meta); err != nil {
		return err
	}

	// Update if we need to
	return resourceAwsInstanceUpdate(d, meta)
}

func resourceAwsInstanceRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	resp, err := ec2conn.DescribeInstances(&ec2.DescribeInstancesRequest{
		InstanceIDs: []string{d.Id()},
	})
	if err != nil {
		// If the instance was not found, return nil so that we can show
		// that the instance is gone.
		if ec2err, ok := err.(aws.APIError); ok && ec2err.Code == "InvalidInstanceID.NotFound" {
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

	instance := &resp.Reservations[0].Instances[0]

	// If the instance is terminated, then it is gone
	if *instance.State.Name == "terminated" {
		d.SetId("")
		return nil
	}

	d.Set("availability_zone", instance.Placement.AvailabilityZone)
	d.Set("key_name", instance.KeyName)
	d.Set("public_dns", instance.PublicDNSName)
	d.Set("public_ip", instance.PublicIPAddress)
	d.Set("private_dns", instance.PrivateDNSName)
	d.Set("private_ip", instance.PrivateIPAddress)
	d.Set("subnet_id", instance.SubnetID)
	if len(instance.NetworkInterfaces) > 0 {
		d.Set("subnet_id", instance.NetworkInterfaces[0].SubnetID)
	} else {
		d.Set("subnet_id", instance.SubnetID)
	}
	d.Set("ebs_optimized", instance.EBSOptimized)
	d.Set("tags", tagsToMap(instance.Tags))
	d.Set("tenancy", instance.Placement.Tenancy)

	// Determine whether we're referring to security groups with
	// IDs or names. We use a heuristic to figure this out. By default,
	// we use IDs if we're in a VPC. However, if we previously had an
	// all-name list of security groups, we use names. Or, if we had any
	// IDs, we use IDs.
	useID := *instance.SubnetID != ""
	if v := d.Get("security_groups"); v != nil {
		match := false
		for _, v := range v.(*schema.Set).List() {
			if strings.HasPrefix(v.(string), "sg-") {
				match = true
				break
			}
		}

		useID = match
	}

	// Build up the security groups
	sgs := make([]string, len(instance.SecurityGroups))
	for i, sg := range instance.SecurityGroups {
		if useID {
			sgs[i] = *sg.GroupID
		} else {
			sgs[i] = *sg.GroupName
		}
	}
	d.Set("security_groups", sgs)

	blockDevices := make(map[string]ec2.InstanceBlockDeviceMapping)
	for _, bd := range instance.BlockDeviceMappings {
		blockDevices[*bd.EBS.VolumeID] = bd
	}

	volIDs := make([]string, 0, len(blockDevices))
	for _, vol := range blockDevices {
		volIDs = append(volIDs, *vol.EBS.VolumeID)
	}

	volResp, err := ec2conn.DescribeVolumes(&ec2.DescribeVolumesRequest{
		VolumeIDs: volIDs,
	})
	if err != nil {
		return err
	}

	nonRootBlockDevices := make([]map[string]interface{}, 0)
	rootBlockDevice := make([]interface{}, 0, 1)
	for _, vol := range volResp.Volumes {
		blockDevice := make(map[string]interface{})
		blockDevice["device_name"] = *blockDevices[*vol.VolumeID].DeviceName
		blockDevice["volume_type"] = *vol.VolumeType
		blockDevice["volume_size"] = *vol.Size
		if vol.IOPS != nil {
			blockDevice["iops"] = *vol.IOPS
		}
		blockDevice["delete_on_termination"] =
			*blockDevices[*vol.VolumeID].EBS.DeleteOnTermination

		// If this is the root device, save it. We stop here since we
		// can't put invalid keys into this map.
		if blockDevice["device_name"] == *instance.RootDeviceName {
			rootBlockDevice = []interface{}{blockDevice}
			continue
		}

		blockDevice["snapshot_id"] = *vol.SnapshotID
		blockDevice["encrypted"] = *vol.Encrypted
		nonRootBlockDevices = append(nonRootBlockDevices, blockDevice)
	}
	d.Set("block_device", nonRootBlockDevices)
	d.Set("root_block_device", rootBlockDevice)

	return nil
}

func resourceAwsInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn
	opts := new(ec2.ModifyInstanceAttributeRequest)

	log.Printf("[INFO] Modifying instance %s: %#v", d.Id(), opts)
	err := ec2conn.ModifyInstanceAttribute(&ec2.ModifyInstanceAttributeRequest{
		InstanceID: aws.String(d.Id()),
		SourceDestCheck: &ec2.AttributeBooleanValue{
			Value: aws.Boolean(d.Get("source_dest_check").(bool)),
		},
	})

	if err != nil {
		return err
	}

	// TODO(mitchellh): wait for the attributes we modified to
	// persist the change...

	if err := setTags(ec2conn, d); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	return nil
}

func resourceAwsInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	log.Printf("[INFO] Terminating instance: %s", d.Id())
	req := &ec2.TerminateInstancesRequest{
		InstanceIDs: []string{d.Id()},
	}
	if _, err := ec2conn.TerminateInstances(req); err != nil {
		return fmt.Errorf("Error terminating instance: %s", err)
	}

	log.Printf(
		"[DEBUG] Waiting for instance (%s) to become terminated",
		d.Id())

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending", "running", "shutting-down", "stopped", "stopping"},
		Target:     "terminated",
		Refresh:    InstanceStateRefreshFunc(ec2conn, d.Id()),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for instance (%s) to terminate: %s",
			d.Id(), err)
	}

	d.SetId("")
	return nil
}

// InstanceStateRefreshFunc returns a resource.StateRefreshFunc that is used to watch
// an EC2 instance.
func InstanceStateRefreshFunc(conn *ec2.EC2, instanceID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeInstances(&ec2.DescribeInstancesRequest{
			InstanceIDs: []string{instanceID},
		})
		if err != nil {
			if ec2err, ok := err.(aws.APIError); ok && ec2err.Code == "InvalidInstanceID.NotFound" {
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

		i := &resp.Reservations[0].Instances[0]
		return i, *i.State.Name, nil
	}
}

func resourceAwsInstanceBlockDevicesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["virtual_name"].(string)))
	buf.WriteString(fmt.Sprintf("%t-", m["delete_on_termination"].(bool)))
	return hashcode.String(buf.String())
}
