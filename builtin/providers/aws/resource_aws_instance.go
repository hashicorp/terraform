package aws

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	codaws "github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/gen/ec2"
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
							ForceNew: true,
						},

						"volume_type": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"volume_size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
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
							ForceNew: true,
						},
					},
				},
				Set: resourceAwsInstanceBlockDevicesHash,
			},
		},
	}
}

func resourceAwsInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).codaConn

	// Figure out user data
	userData := ""
	if v := d.Get("user_data"); v != nil {
		userData = v.(string)
	}

	associatePublicIPAddress := false
	if v := d.Get("associate_public_ip_address"); v != nil {
		associatePublicIPAddress = v.(bool)
	}

	netInterfaceSpec := ec2.InstanceNetworkInterfaceSpecification{
		AssociatePublicIPAddress: associatePublicIPAddress,
	}

	placement := ec2.Placement{
		AvailabilityZone: d.Get("availability_zone").(string),
		Tenancy:          d.Get("tenancy").(string),
	}

	iamInstanceProfile := ec2.IamInstanceProfileSpecification{
		Name: d.Get("iam_instance_profile").(string),
	}

	runOpts := ec2.RunInstancesRequest{
		ImageID:            d.Get("ami").(string),
		InstanceType:       d.Get("instance_type").(string),
		KeyName:            d.Get("key_name").(string),
		SubnetID:           d.Get("subnet_id").(string),
		PrivateIPAddress:   d.Get("private_ip").(string),
		UserData:           userData,
		MinCount:           1,
		MaxCount:           1,
		EbsOptimized:       d.Get("ebs_optimized").(bool),
		IamInstanceProfile: iamInstanceProfile,
		Placement:          placement,
		NetworkInterfaces:  []ec2.InstanceNetworkInterfaceSpecification{netInterfaceSpec},
	}

	if v := d.Get("security_groups"); v != nil {
		for _, v := range v.(*schema.Set).List() {
			str := v.(string)
			runOpts.SecurityGroups = append(runOpts.SecurityGroups, str)
		}
	}

	if v := d.Get("block_device"); v != nil {
		vs := v.(*schema.Set).List()
		if len(vs) > 0 {
			runOpts.BlockDeviceMappings = make([]ec2.BlockDeviceMapping, len(vs))
			for i, v := range vs {
				bd := v.(map[string]interface{})

				runOpts.BlockDeviceMappings[i].Ebs = ec2.EbsBlockDevice{
					DeleteOnTermination: bd["delete_on_termination"].(bool),
					Encrypted:           bd["encrypted"].(bool),
					SnapshotID:          bd["snapshot_id"].(string),
					VolumeType:          bd["volume_type"].(string),
					VolumeSize:          bd["volume_size"].(int),
				}
				runOpts.BlockDeviceMappings[i].DeviceName = bd["device_name"].(string)
				runOpts.BlockDeviceMappings[i].VirtualName = bd["virtual_name"].(string)
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
	log.Printf("[INFO] Instance ID: %s", instance.InstanceID)

	// Store the resulting ID so we can look this up later
	d.SetId(instance.InstanceID)

	// Wait for the instance to become running so we can get some attributes
	// that aren't available until later.
	log.Printf(
		"[DEBUG] Waiting for instance (%s) to become running",
		instance.InstanceID)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     "running",
		Refresh:    InstanceStateRefreshFunc(ec2conn, instance.InstanceID),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	instanceRaw, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for instance (%s) to become ready: %s",
			instance.InstanceID, err)
	}

	instance = instanceRaw.(*ec2.Instance)

	// Initialize the connection info
	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": instance.PublicIPAddress,
	})

	// Set our attributes
	if err := resourceAwsInstanceRead(d, meta); err != nil {
		return err
	}

	// Update if we need to
	return resourceAwsInstanceUpdate(d, meta)
}

func resourceAwsInstanceRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).codaConn

	req := ec2.DescribeInstancesRequest{
		InstanceIds: []string{d.Id()},
	}

	resp, err := ec2conn.DescribeInstances(req)

	if err != nil {
		// If the instance was not found, return nil so that we can show
		// that the instance is gone.
		if ec2err, ok := err.(*codaws.APIError); ok && ec2err.Code == "InvalidInstanceID.NotFound" {
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
	if instance.State.Name == "terminated" {
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
	d.Set("ebs_optimized", instance.EbsOptimized)
	d.Set("tenancy", instance.Placement.Tenancy)

	// Determine whether we're referring to security groups with
	// IDs or names. We use a heuristic to figure this out. By default,
	// we use IDs if we're in a VPC. However, if we previously had an
	// all-name list of security groups, we use names. Or, if we had any
	// IDs, we use IDs.
	useID := instance.SubnetID != ""
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
			sgs[i] = sg.GroupID
		} else {
			sgs[i] = sg.GroupName
		}
	}
	d.Set("security_groups", sgs)

	return nil
}

func resourceAwsInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).codaConn

	modify := false
	opts := ec2.ModifyInstanceAttributeRequest{
		InstanceID: d.Id(),
	}

	if v, ok := d.GetOk("source_dest_check"); ok {
		opts.SourceDestCheck = ec2.AttributeBooleanValue{
			Value: v.(bool),
		}
		modify = true
	}

	if modify {
		log.Printf("[INFO] Modifing instance %s: %#v", d.Id(), opts)
		if err := ec2conn.ModifyInstanceAttribute(opts); err != nil {
			return err
		}

		// TODO(mitchellh): wait for the attributes we modified to
		// persist the change...
	}

	return nil
}

func resourceAwsInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).codaConn

	opts := ec2.TerminateInstancesRequest{
		InstanceIds: []string{d.Id()},
	}

	log.Printf("[INFO] Terminating instance: %s", d.Id())
	if _, err := ec2conn.TerminateInstances(opts); err != nil {
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
		req := ec2.DescribeInstancesRequest{
			InstanceIds: []string{instanceID},
		}

		resp, err := conn.DescribeInstances(req)

		if err != nil {
			if ec2err, ok := err.(*codaws.APIError); ok && ec2err.Code == "InvalidInstanceID.NotFound" {
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
		return i, i.State.Name, nil
	}
}

func resourceAwsInstanceBlockDevicesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	buf.WriteString(fmt.Sprintf("%s-", m["device_name"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["snapshot_id"].(string)))
	buf.WriteString(fmt.Sprintf("%s-", m["volume_type"].(string)))
	buf.WriteString(fmt.Sprintf("%d-", m["volume_size"].(int)))
	buf.WriteString(fmt.Sprintf("%t-", m["delete_on_termination"].(bool)))
	buf.WriteString(fmt.Sprintf("%t-", m["encrypted"].(bool)))
	return hashcode.String(buf.String())
}
