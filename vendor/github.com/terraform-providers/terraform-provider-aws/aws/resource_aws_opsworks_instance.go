package aws

import (
	"bytes"
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/opsworks"
)

func resourceAwsOpsworksInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsOpsworksInstanceCreate,
		Read:   resourceAwsOpsworksInstanceRead,
		Update: resourceAwsOpsworksInstanceUpdate,
		Delete: resourceAwsOpsworksInstanceDelete,
		Importer: &schema.ResourceImporter{
			State: resourceAwsOpsworksInstanceImport,
		},

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(10 * time.Minute),
			Delete: schema.DefaultTimeout(10 * time.Minute),
			Update: schema.DefaultTimeout(10 * time.Minute),
		},

		Schema: map[string]*schema.Schema{
			"agent_version": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "INHERIT",
			},

			"ami_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"architecture": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "x86_64",
				ValidateFunc: validation.StringInSlice([]string{
					opsworks.ArchitectureX8664,
					opsworks.ArchitectureI386,
				}, false),
			},

			"auto_scaling_type": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					opsworks.AutoScalingTypeLoad,
					opsworks.AutoScalingTypeTimer,
				}, false),
			},

			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"created_at": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"delete_ebs": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"delete_eip": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"ebs_optimized": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"ec2_instance_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ecs_cluster_arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"elastic_ip": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"hostname": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"infrastructure_class": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"install_updates_on_boot": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"instance_profile_arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"instance_type": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"last_service_error_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"layer_ids": {
				Type:     schema.TypeList,
				Required: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"os": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"platform": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"private_dns": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"private_ip": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"public_dns": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"public_ip": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"registered_by": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"reported_agent_version": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"reported_os_family": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"reported_os_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"reported_os_version": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"root_device_type": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
				ValidateFunc: validation.StringInSlice([]string{
					opsworks.RootDeviceTypeEbs,
					opsworks.RootDeviceTypeInstanceStore,
				}, false),
			},

			"root_device_volume_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"security_group_ids": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"ssh_host_dsa_key_fingerprint": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"ssh_host_rsa_key_fingerprint": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"ssh_key_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"stack_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"state": {
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: validation.StringInSlice([]string{
					"running",
					"stopped",
				}, false),
			},

			"status": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"subnet_id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"tenancy": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					"dedicated",
					"default",
					"host",
				}, false),
			},

			"virtualization_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
				ValidateFunc: validation.StringInSlice([]string{
					opsworks.VirtualizationTypeParavirtual,
					opsworks.VirtualizationTypeHvm,
				}, false),
			},

			"ebs_block_device": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
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

						"iops": {
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
							ForceNew: true,
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

			"root_block_device": {
				// TODO: This is a set because we don't support singleton
				//       sub-resources today. We'll enforce that the set only ever has
				//       length zero or one below. When TF gains support for
				//       sub-resources this can be converted.
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
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
							Type:     schema.TypeInt,
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

func resourceAwsOpsworksInstanceValidate(d *schema.ResourceData) error {
	if d.HasChange("ami_id") {
		if v, ok := d.GetOk("os"); ok {
			if v.(string) != "Custom" {
				return fmt.Errorf("OS must be \"Custom\" when using using a custom ami_id")
			}
		}

		if _, ok := d.GetOk("root_block_device"); ok {
			return fmt.Errorf("Cannot specify root_block_device when using a custom ami_id.")
		}

		if _, ok := d.GetOk("ebs_block_device"); ok {
			return fmt.Errorf("Cannot specify ebs_block_device when using a custom ami_id.")
		}

		if _, ok := d.GetOk("ephemeral_block_device"); ok {
			return fmt.Errorf("Cannot specify ephemeral_block_device when using a custom ami_id.")
		}
	}
	return nil
}

func resourceAwsOpsworksInstanceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	req := &opsworks.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(d.Id()),
		},
	}

	log.Printf("[DEBUG] Reading OpsWorks instance: %s", d.Id())

	resp, err := client.DescribeInstances(req)
	if err != nil {
		if awserr, ok := err.(awserr.Error); ok {
			if awserr.Code() == "ResourceNotFoundException" {
				d.SetId("")
				return nil
			}
		}
		return err
	}

	// If nothing was found, then return no state
	if len(resp.Instances) == 0 {
		d.SetId("")
		return nil
	}
	instance := resp.Instances[0]

	if instance.InstanceId == nil {
		d.SetId("")
		return nil
	}
	instanceId := *instance.InstanceId

	d.SetId(instanceId)
	d.Set("agent_version", instance.AgentVersion)
	d.Set("ami_id", instance.AmiId)
	d.Set("architecture", instance.Architecture)
	d.Set("auto_scaling_type", instance.AutoScalingType)
	d.Set("availability_zone", instance.AvailabilityZone)
	d.Set("created_at", instance.CreatedAt)
	d.Set("ebs_optimized", instance.EbsOptimized)
	d.Set("ec2_instance_id", instance.Ec2InstanceId)
	d.Set("ecs_cluster_arn", instance.EcsClusterArn)
	d.Set("elastic_ip", instance.ElasticIp)
	d.Set("hostname", instance.Hostname)
	d.Set("infrastructure_class", instance.InfrastructureClass)
	d.Set("install_updates_on_boot", instance.InstallUpdatesOnBoot)
	d.Set("instance_profile_arn", instance.InstanceProfileArn)
	d.Set("instance_type", instance.InstanceType)
	d.Set("last_service_error_id", instance.LastServiceErrorId)
	var layerIds []string
	for _, v := range instance.LayerIds {
		layerIds = append(layerIds, *v)
	}
	layerIds, err = sortListBasedonTFFile(layerIds, d, "layer_ids")
	if err != nil {
		return fmt.Errorf("Error sorting layer_ids attribute: %#v", err)
	}
	if err := d.Set("layer_ids", layerIds); err != nil {
		return fmt.Errorf("Error setting layer_ids attribute: %#v, error: %#v", layerIds, err)
	}
	d.Set("os", instance.Os)
	d.Set("platform", instance.Platform)
	d.Set("private_dns", instance.PrivateDns)
	d.Set("private_ip", instance.PrivateIp)
	d.Set("public_dns", instance.PublicDns)
	d.Set("public_ip", instance.PublicIp)
	d.Set("registered_by", instance.RegisteredBy)
	d.Set("reported_agent_version", instance.ReportedAgentVersion)
	d.Set("reported_os_family", instance.ReportedOs.Family)
	d.Set("reported_os_name", instance.ReportedOs.Name)
	d.Set("reported_os_version", instance.ReportedOs.Version)
	d.Set("root_device_type", instance.RootDeviceType)
	d.Set("root_device_volume_id", instance.RootDeviceVolumeId)
	d.Set("ssh_host_dsa_key_fingerprint", instance.SshHostDsaKeyFingerprint)
	d.Set("ssh_host_rsa_key_fingerprint", instance.SshHostRsaKeyFingerprint)
	d.Set("ssh_key_name", instance.SshKeyName)
	d.Set("stack_id", instance.StackId)
	d.Set("status", instance.Status)
	d.Set("subnet_id", instance.SubnetId)
	d.Set("tenancy", instance.Tenancy)
	d.Set("virtualization_type", instance.VirtualizationType)

	// Read BlockDeviceMapping
	ibds, err := readOpsworksBlockDevices(d, instance, meta)
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

	// Read Security Groups
	sgs := make([]string, 0, len(instance.SecurityGroupIds))
	for _, sg := range instance.SecurityGroupIds {
		sgs = append(sgs, *sg)
	}
	if err := d.Set("security_group_ids", sgs); err != nil {
		return err
	}

	return nil
}

func resourceAwsOpsworksInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	err := resourceAwsOpsworksInstanceValidate(d)
	if err != nil {
		return err
	}

	req := &opsworks.CreateInstanceInput{
		AgentVersion:         aws.String(d.Get("agent_version").(string)),
		Architecture:         aws.String(d.Get("architecture").(string)),
		EbsOptimized:         aws.Bool(d.Get("ebs_optimized").(bool)),
		InstallUpdatesOnBoot: aws.Bool(d.Get("install_updates_on_boot").(bool)),
		InstanceType:         aws.String(d.Get("instance_type").(string)),
		LayerIds:             expandStringList(d.Get("layer_ids").([]interface{})),
		StackId:              aws.String(d.Get("stack_id").(string)),
	}

	if v, ok := d.GetOk("ami_id"); ok {
		req.AmiId = aws.String(v.(string))
		req.Os = aws.String("Custom")
	}

	if v, ok := d.GetOk("auto_scaling_type"); ok {
		req.AutoScalingType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("availability_zone"); ok {
		req.AvailabilityZone = aws.String(v.(string))
	}

	if v, ok := d.GetOk("hostname"); ok {
		req.Hostname = aws.String(v.(string))
	}

	if v, ok := d.GetOk("os"); ok {
		req.Os = aws.String(v.(string))
	}

	if v, ok := d.GetOk("root_device_type"); ok {
		req.RootDeviceType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("ssh_key_name"); ok {
		req.SshKeyName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("subnet_id"); ok {
		req.SubnetId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("tenancy"); ok {
		req.Tenancy = aws.String(v.(string))
	}

	if v, ok := d.GetOk("virtualization_type"); ok {
		req.VirtualizationType = aws.String(v.(string))
	}

	var blockDevices []*opsworks.BlockDeviceMapping

	if v, ok := d.GetOk("ebs_block_device"); ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			ebs := &opsworks.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(bd["delete_on_termination"].(bool)),
			}

			if v, ok := bd["snapshot_id"].(string); ok && v != "" {
				ebs.SnapshotId = aws.String(v)
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

			blockDevices = append(blockDevices, &opsworks.BlockDeviceMapping{
				DeviceName: aws.String(bd["device_name"].(string)),
				Ebs:        ebs,
			})
		}
	}

	if v, ok := d.GetOk("ephemeral_block_device"); ok {
		vL := v.(*schema.Set).List()
		for _, v := range vL {
			bd := v.(map[string]interface{})
			blockDevices = append(blockDevices, &opsworks.BlockDeviceMapping{
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
			ebs := &opsworks.EbsBlockDevice{
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

			blockDevices = append(blockDevices, &opsworks.BlockDeviceMapping{
				DeviceName: aws.String("ROOT_DEVICE"),
				Ebs:        ebs,
			})
		}
	}

	if len(blockDevices) > 0 {
		req.BlockDeviceMappings = blockDevices
	}

	log.Printf("[DEBUG] Creating OpsWorks instance")

	var resp *opsworks.CreateInstanceOutput

	resp, err = client.CreateInstance(req)
	if err != nil {
		return err
	}

	if resp.InstanceId == nil {
		return fmt.Errorf("Error launching instance: no instance returned in response")
	}

	instanceId := *resp.InstanceId
	d.SetId(instanceId)

	if v, ok := d.GetOk("state"); ok && v.(string) == "running" {
		err := startOpsworksInstance(d, meta, true, d.Timeout(schema.TimeoutCreate))
		if err != nil {
			return err
		}
	}

	return resourceAwsOpsworksInstanceRead(d, meta)
}

func resourceAwsOpsworksInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	err := resourceAwsOpsworksInstanceValidate(d)
	if err != nil {
		return err
	}

	req := &opsworks.UpdateInstanceInput{
		InstanceId:           aws.String(d.Id()),
		AgentVersion:         aws.String(d.Get("agent_version").(string)),
		Architecture:         aws.String(d.Get("architecture").(string)),
		InstallUpdatesOnBoot: aws.Bool(d.Get("install_updates_on_boot").(bool)),
	}

	if v, ok := d.GetOk("ami_id"); ok {
		req.AmiId = aws.String(v.(string))
		req.Os = aws.String("Custom")
	}

	if v, ok := d.GetOk("auto_scaling_type"); ok {
		req.AutoScalingType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("hostname"); ok {
		req.Hostname = aws.String(v.(string))
	}

	if v, ok := d.GetOk("instance_type"); ok {
		req.InstanceType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("layer_ids"); ok {
		req.LayerIds = expandStringList(v.([]interface{}))
	}

	if v, ok := d.GetOk("os"); ok {
		req.Os = aws.String(v.(string))
	}

	if v, ok := d.GetOk("ssh_key_name"); ok {
		req.SshKeyName = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Updating OpsWorks instance: %s", d.Id())

	_, err = client.UpdateInstance(req)
	if err != nil {
		return err
	}

	var status string

	if v, ok := d.GetOk("status"); ok {
		status = v.(string)
	} else {
		status = "stopped"
	}

	if v, ok := d.GetOk("state"); ok {
		state := v.(string)
		if state == "running" {
			if status == "stopped" || status == "stopping" || status == "shutting_down" {
				err := startOpsworksInstance(d, meta, false, d.Timeout(schema.TimeoutUpdate))
				if err != nil {
					return err
				}
			}
		} else {
			if status != "stopped" && status != "stopping" && status != "shutting_down" {
				err := stopOpsworksInstance(d, meta, true, d.Timeout(schema.TimeoutUpdate))
				if err != nil {
					return err
				}
			}
		}
	}

	return resourceAwsOpsworksInstanceRead(d, meta)
}

func resourceAwsOpsworksInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AWSClient).opsworksconn

	if v, ok := d.GetOk("status"); ok && v.(string) != "stopped" {
		err := stopOpsworksInstance(d, meta, true, d.Timeout(schema.TimeoutDelete))
		if err != nil {
			return err
		}
	}

	req := &opsworks.DeleteInstanceInput{
		InstanceId:      aws.String(d.Id()),
		DeleteElasticIp: aws.Bool(d.Get("delete_eip").(bool)),
		DeleteVolumes:   aws.Bool(d.Get("delete_ebs").(bool)),
	}

	log.Printf("[DEBUG] Deleting OpsWorks instance: %s", d.Id())

	_, err := client.DeleteInstance(req)
	if err != nil {
		return err
	}

	return nil
}

func resourceAwsOpsworksInstanceImport(
	d *schema.ResourceData, meta interface{}) ([]*schema.ResourceData, error) {
	// Neither delete_eip nor delete_ebs can be fetched
	// from any API call, so we need to default to the values
	// we set in the schema by default
	d.Set("delete_ebs", true)
	d.Set("delete_eip", true)
	return []*schema.ResourceData{d}, nil
}

func startOpsworksInstance(d *schema.ResourceData, meta interface{}, wait bool, timeout time.Duration) error {
	client := meta.(*AWSClient).opsworksconn

	instanceId := d.Id()

	req := &opsworks.StartInstanceInput{
		InstanceId: aws.String(instanceId),
	}

	log.Printf("[DEBUG] Starting OpsWorks instance: %s", instanceId)

	_, err := client.StartInstance(req)

	if err != nil {
		return err
	}

	if wait {
		log.Printf("[DEBUG] Waiting for instance (%s) to become running", instanceId)

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"requested", "pending", "booting", "running_setup"},
			Target:     []string{"online"},
			Refresh:    OpsworksInstanceStateRefreshFunc(client, instanceId),
			Timeout:    timeout,
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}
		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for instance (%s) to become stopped: %s",
				instanceId, err)
		}
	}

	return nil
}

func stopOpsworksInstance(d *schema.ResourceData, meta interface{}, wait bool, timeout time.Duration) error {
	client := meta.(*AWSClient).opsworksconn

	instanceId := d.Id()

	req := &opsworks.StopInstanceInput{
		InstanceId: aws.String(instanceId),
	}

	log.Printf("[DEBUG] Stopping OpsWorks instance: %s", instanceId)

	_, err := client.StopInstance(req)

	if err != nil {
		return err
	}

	if wait {
		log.Printf("[DEBUG] Waiting for instance (%s) to become stopped", instanceId)

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"stopping", "terminating", "shutting_down", "terminated"},
			Target:     []string{"stopped"},
			Refresh:    OpsworksInstanceStateRefreshFunc(client, instanceId),
			Timeout:    timeout,
			Delay:      10 * time.Second,
			MinTimeout: 3 * time.Second,
		}
		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for instance (%s) to become stopped: %s",
				instanceId, err)
		}
	}

	return nil
}

func readOpsworksBlockDevices(d *schema.ResourceData, instance *opsworks.Instance, meta interface{}) (
	map[string]interface{}, error) {

	blockDevices := make(map[string]interface{})
	blockDevices["ebs"] = make([]map[string]interface{}, 0)
	blockDevices["ephemeral"] = make([]map[string]interface{}, 0)
	blockDevices["root"] = nil

	if len(instance.BlockDeviceMappings) == 0 {
		return nil, nil
	}

	for _, bdm := range instance.BlockDeviceMappings {
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
		if bdm.DeviceName != nil && *bdm.DeviceName == "ROOT_DEVICE" {
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

func OpsworksInstanceStateRefreshFunc(conn *opsworks.OpsWorks, instanceID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeInstances(&opsworks.DescribeInstancesInput{
			InstanceIds: []*string{aws.String(instanceID)},
		})
		if err != nil {
			if awserr, ok := err.(awserr.Error); ok && awserr.Code() == "ResourceNotFoundException" {
				// Set this to nil as if we didn't find anything.
				resp = nil
			} else {
				log.Printf("Error on OpsworksInstanceStateRefresh: %s", err)
				return nil, "", err
			}
		}

		if resp == nil || len(resp.Instances) == 0 {
			// Sometimes AWS just has consistency issues and doesn't see
			// our instance yet. Return an empty state.
			return nil, "", nil
		}

		i := resp.Instances[0]
		return i, *i.Status, nil
	}
}
