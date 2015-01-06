package aws

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/mitchellh/goamz/ec2"
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
					},
				},
				Set: resourceAwsInstanceBlockDevicesHash,
			},
		},
	}
}

func resourceAwsInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	// Figure out user data
	userData := ""
	if v := d.Get("user_data"); v != nil {
		userData = v.(string)
	}

	associatePublicIPAddress := false
	if v := d.Get("associate_public_ip_address"); v != nil {
		associatePublicIPAddress = v.(bool)
	}

	// Build the creation struct
	runOpts := &ec2.RunInstances{
		ImageId:                  d.Get("ami").(string),
		AvailZone:                d.Get("availability_zone").(string),
		InstanceType:             d.Get("instance_type").(string),
		KeyName:                  d.Get("key_name").(string),
		SubnetId:                 d.Get("subnet_id").(string),
		PrivateIPAddress:         d.Get("private_ip").(string),
		AssociatePublicIpAddress: associatePublicIPAddress,
		UserData:                 []byte(userData),
		EbsOptimized:             d.Get("ebs_optimized").(bool),
		IamInstanceProfile:       d.Get("iam_instance_profile").(string),
		Tenancy:                  d.Get("tenancy").(string),
	}

	if v := d.Get("security_groups"); v != nil {
		for _, v := range v.(*schema.Set).List() {
			str := v.(string)

			var g ec2.SecurityGroup
			if runOpts.SubnetId != "" {
				g.Id = str
			} else {
				g.Name = str
			}

			runOpts.SecurityGroups = append(runOpts.SecurityGroups, g)
		}
	}

	if v := d.Get("block_device"); v != nil {
		vs := v.(*schema.Set).List()
		if len(vs) > 0 {
			runOpts.BlockDevices = make([]ec2.BlockDeviceMapping, len(vs))
			for i, v := range vs {
				bd := v.(map[string]interface{})
				runOpts.BlockDevices[i].DeviceName = bd["device_name"].(string)
				runOpts.BlockDevices[i].VirtualName = bd["virtual_name"].(string)
				runOpts.BlockDevices[i].SnapshotId = bd["snapshot_id"].(string)
				runOpts.BlockDevices[i].VolumeType = bd["volume_type"].(string)
				runOpts.BlockDevices[i].VolumeSize = int64(bd["volume_size"].(int))
				runOpts.BlockDevices[i].DeleteOnTermination = bd["delete_on_termination"].(bool)
				runOpts.BlockDevices[i].Encrypted = bd["encrypted"].(bool)
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
	log.Printf("[INFO] Instance ID: %s", instance.InstanceId)

	// Store the resulting ID so we can look this up later
	d.SetId(instance.InstanceId)

	// Wait for the instance to become running so we can get some attributes
	// that aren't available until later.
	log.Printf(
		"[DEBUG] Waiting for instance (%s) to become running",
		instance.InstanceId)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     "running",
		Refresh:    InstanceStateRefreshFunc(ec2conn, instance.InstanceId),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	instanceRaw, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for instance (%s) to become ready: %s",
			instance.InstanceId, err)
	}

	instance = instanceRaw.(*ec2.Instance)

	// Initialize the connection info
	d.SetConnInfo(map[string]string{
		"type": "ssh",
		"host": instance.PublicIpAddress,
	})

	// Set our attributes
	if err := resourceAwsInstanceRead(d, meta); err != nil {
		return err
	}

	// Update if we need to
	return resourceAwsInstanceUpdate(d, meta)
}

func resourceAwsInstanceRead(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	resp, err := ec2conn.Instances([]string{d.Id()}, ec2.NewFilter())
	if err != nil {
		// If the instance was not found, return nil so that we can show
		// that the instance is gone.
		if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidInstanceID.NotFound" {
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

	d.Set("availability_zone", instance.AvailZone)
	d.Set("key_name", instance.KeyName)
	d.Set("public_dns", instance.DNSName)
	d.Set("public_ip", instance.PublicIpAddress)
	d.Set("private_dns", instance.PrivateDNSName)
	d.Set("private_ip", instance.PrivateIpAddress)
	d.Set("subnet_id", instance.SubnetId)
	d.Set("ebs_optimized", instance.EbsOptimized)
	d.Set("tags", tagsToMap(instance.Tags))
	d.Set("tenancy", instance.Tenancy)

	// Determine whether we're referring to security groups with
	// IDs or names. We use a heuristic to figure this out. By default,
	// we use IDs if we're in a VPC. However, if we previously had an
	// all-name list of security groups, we use names. Or, if we had any
	// IDs, we use IDs.
	useID := instance.SubnetId != ""
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
			sgs[i] = sg.Id
		} else {
			sgs[i] = sg.Name
		}
	}

	d.Set("security_groups", sgs)

	volIDs := make([]string, len(instance.BlockDevices))
	bdByVolID := make(map[string]ec2.BlockDevice)
	for i, bd := range instance.BlockDevices {
		volIDs[i] = bd.VolumeId
		bdByVolID[bd.VolumeId] = bd
	}

	volResp, err := ec2conn.Volumes(volIDs, ec2.NewFilter())
	if err != nil {
		return err
	}

	bds := make([]map[string]interface{}, len(volResp.Volumes))
	for i, vol := range volResp.Volumes {
		volSize, err := strconv.Atoi(vol.Size)
		if err != nil {
			return err
		}
		bds[i] = make(map[string]interface{})
		bds[i]["device_name"] = bdByVolID[vol.VolumeId].DeviceName
		bds[i]["snapshot_id"] = vol.SnapshotId
		bds[i]["volume_type"] = vol.VolumeType
		bds[i]["volume_size"] = volSize
		bds[i]["delete_on_termination"] = bdByVolID[vol.VolumeId].DeleteOnTermination
		bds[i]["encrypted"] = vol.Encrypted
	}
	d.Set("block_device", bds)

	return nil
}

func resourceAwsInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	ec2conn := meta.(*AWSClient).ec2conn

	// We might need to do several calls (http://goo.gl/g3glRV):
	// "Modifies the specified attribute of the specified instance. You can specify
	// only one attribute at a time."

	s := []*ec2.ModifyInstance{}

	if v, ok := d.GetOk("source_dest_check"); ok {
		opts := new(ec2.ModifyInstance)
		opts.SourceDestCheck = v.(bool)
		opts.SetSourceDestCheck = true
		s = append(s, opts)
	}

	if v := d.Get("security_groups"); v != nil {
		opts := new(ec2.ModifyInstance)
		for _, v := range v.(*schema.Set).List() {
			str := v.(string)

			var g ec2.SecurityGroup
			g.Id = str
			opts.SecurityGroups = append(opts.SecurityGroups, g)
		}
		s = append(s, opts)
	}

	for _, v := range s {
		log.Printf("[INFO] Modifing instance %s: %#v", d.Id(), v)
		if _, err := ec2conn.ModifyInstance(d.Id(), v); err != nil {
			return err
		}

		// TODO(mitchellh): wait for the attributes we modified to
		// persist the change...
	}

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
	if _, err := ec2conn.TerminateInstances([]string{d.Id()}); err != nil {
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
		resp, err := conn.Instances([]string{instanceID}, ec2.NewFilter())
		if err != nil {
			if ec2err, ok := err.(*ec2.Error); ok && ec2err.Code == "InvalidInstanceID.NotFound" {
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
	buf.WriteString(fmt.Sprintf("%s-", m["virtual_name"].(string)))
	buf.WriteString(fmt.Sprintf("%t-", m["delete_on_termination"].(bool)))
	return hashcode.String(buf.String())
}
