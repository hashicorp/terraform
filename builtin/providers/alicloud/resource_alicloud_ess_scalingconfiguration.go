package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/denverdino/aliyungo/ess"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
	"time"
)

func resourceAlicloudEssScalingConfiguration() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunEssScalingConfigurationCreate,
		Read:   resourceAliyunEssScalingConfigurationRead,
		Update: resourceAliyunEssScalingConfigurationUpdate,
		Delete: resourceAliyunEssScalingConfigurationDelete,

		Schema: map[string]*schema.Schema{
			"active": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"enable": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
			},
			"scaling_group_id": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"image_id": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"instance_type": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"io_optimized": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateIoOptimized,
			},
			"security_group_id": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"scaling_configuration_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"internet_charge_type": &schema.Schema{
				Type:         schema.TypeString,
				ForceNew:     true,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateInternetChargeType,
			},
			"internet_max_bandwidth_in": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},
			"internet_max_bandwidth_out": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateInternetMaxBandWidthOut,
			},
			"system_disk_category": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
				ValidateFunc: validateAllowedStringValue([]string{
					string(ecs.DiskCategoryCloud),
					string(ecs.DiskCategoryCloudSSD),
					string(ecs.DiskCategoryCloudEfficiency),
					string(ecs.DiskCategoryEphemeralSSD),
				}),
			},
			"data_disk": &schema.Schema{
				Optional: true,
				ForceNew: true,
				Type:     schema.TypeList,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"size": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
						},
						"category": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"snapshot_id": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
						"device": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
			"instance_ids": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				MaxItems: 20,
			},
		},
	}
}

func resourceAliyunEssScalingConfigurationCreate(d *schema.ResourceData, meta interface{}) error {

	args, err := buildAlicloudEssScalingConfigurationArgs(d, meta)
	if err != nil {
		return err
	}

	essconn := meta.(*AliyunClient).essconn

	scaling, err := essconn.CreateScalingConfiguration(args)
	if err != nil {
		return err
	}

	d.SetId(d.Get("scaling_group_id").(string) + COLON_SEPARATED + scaling.ScalingConfigurationId)

	return resourceAliyunEssScalingConfigurationUpdate(d, meta)
}

func resourceAliyunEssScalingConfigurationUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	if d.HasChange("active") {
		active := d.Get("active").(bool)
		if !active {
			return fmt.Errorf("Please active the scaling configuration directly.")
		}
		ids := strings.Split(d.Id(), COLON_SEPARATED)
		err := client.ActiveScalingConfigurationById(ids[0], ids[1])

		if err != nil {
			return fmt.Errorf("Active scaling configuration %s err: %#v", ids[1], err)
		}
	}

	if err := enableEssScalingConfiguration(d, meta); err != nil {
		return err
	}

	return resourceAliyunEssScalingConfigurationRead(d, meta)
}

func enableEssScalingConfiguration(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	ids := strings.Split(d.Id(), COLON_SEPARATED)

	if d.HasChange("enable") {
		d.SetPartial("enable")
		enable := d.Get("enable").(bool)
		if !enable {
			err := client.DisableScalingConfigurationById(ids[0])

			if err != nil {
				return fmt.Errorf("Disable scaling group %s err: %#v", ids[0], err)
			}
		}

		instance_ids := []string{}
		if d.HasChange("instance_ids") {
			d.SetPartial("instance_ids")
			instances := d.Get("instance_ids").([]interface{})
			instance_ids = expandStringList(instances)
		}
		err := client.EnableScalingConfigurationById(ids[0], ids[1], instance_ids)

		if err != nil {
			return fmt.Errorf("Enable scaling configuration %s err: %#v", ids[1], err)
		}
	}
	return nil
}

func resourceAliyunEssScalingConfigurationRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*AliyunClient)
	ids := strings.Split(d.Id(), COLON_SEPARATED)
	c, err := client.DescribeScalingConfigurationById(ids[0], ids[1])
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == InstanceNotfound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error Describe ESS scaling configuration Attribute: %#v", err)
	}

	d.Set("scaling_group_id", c.ScalingGroupId)
	d.Set("active", c.LifecycleState == ess.Active)
	d.Set("image_id", c.ImageId)
	d.Set("instance_type", c.InstanceType)
	d.Set("io_optimized", c.IoOptimized)
	d.Set("security_group_id", c.SecurityGroupId)
	d.Set("scaling_configuration_name", c.ScalingConfigurationName)
	d.Set("internet_charge_type", c.InternetChargeType)
	d.Set("internet_max_bandwidth_in", c.InternetMaxBandwidthIn)
	d.Set("internet_max_bandwidth_out", c.InternetMaxBandwidthOut)
	d.Set("system_disk_category", c.SystemDiskCategory)
	d.Set("data_disk", flattenDataDiskMappings(c.DataDisks.DataDisk))

	return nil
}

func resourceAliyunEssScalingConfigurationDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)

	return resource.Retry(5*time.Minute, func() *resource.RetryError {
		ids := strings.Split(d.Id(), COLON_SEPARATED)
		err := client.DeleteScalingConfigurationById(ids[0], ids[1])

		if err != nil {
			e, _ := err.(*common.Error)
			if e.ErrorResponse.Code == IncorrectScalingConfigurationLifecycleState {
				return resource.NonRetryableError(
					fmt.Errorf("Scaling configuration is active - please active another one and trying again."))
			}
			if e.ErrorResponse.Code != InvalidScalingGroupIdNotFound {
				return resource.RetryableError(
					fmt.Errorf("Scaling configuration in use - trying again while it is deleted."))
			}
		}

		_, err = client.DescribeScalingConfigurationById(ids[0], ids[1])
		if err != nil {
			if notFoundError(err) {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(
			fmt.Errorf("Scaling configuration in use - trying again while it is deleted."))
	})
}

func buildAlicloudEssScalingConfigurationArgs(d *schema.ResourceData, meta interface{}) (*ess.CreateScalingConfigurationArgs, error) {
	args := &ess.CreateScalingConfigurationArgs{
		ScalingGroupId:  d.Get("scaling_group_id").(string),
		ImageId:         d.Get("image_id").(string),
		InstanceType:    d.Get("instance_type").(string),
		IoOptimized:     ecs.IoOptimized(d.Get("io_optimized").(string)),
		SecurityGroupId: d.Get("security_group_id").(string),
	}

	if v := d.Get("scaling_configuration_name").(string); v != "" {
		args.ScalingConfigurationName = v
	}

	if v := d.Get("internet_charge_type").(string); v != "" {
		args.InternetChargeType = common.InternetChargeType(v)
	}

	if v := d.Get("internet_max_bandwidth_in").(int); v != 0 {
		args.InternetMaxBandwidthIn = v
	}

	if v := d.Get("internet_max_bandwidth_out").(int); v != 0 {
		args.InternetMaxBandwidthOut = v
	}

	if v := d.Get("system_disk_category").(string); v != "" {
		args.SystemDisk_Category = common.UnderlineString(v)
	}

	dds, ok := d.GetOk("data_disk")
	if ok {
		disks := dds.([]interface{})
		diskTypes := []ess.DataDiskType{}

		for _, e := range disks {
			pack := e.(map[string]interface{})
			disk := ess.DataDiskType{
				Size:       pack["size"].(int),
				Category:   pack["category"].(string),
				SnapshotId: pack["snapshot_id"].(string),
				Device:     pack["device"].(string),
			}
			if v := pack["size"].(int); v != 0 {
				disk.Size = v
			}
			if v := pack["category"].(string); v != "" {
				disk.Category = v
			}
			if v := pack["snapshot_id"].(string); v != "" {
				disk.SnapshotId = v
			}
			if v := pack["device"].(string); v != "" {
				disk.Device = v
			}
			diskTypes = append(diskTypes, disk)
		}
		args.DataDisk = diskTypes
	}

	return args, nil
}
