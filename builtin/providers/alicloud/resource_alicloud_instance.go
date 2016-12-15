package alicloud

import (
	"fmt"
	"log"

	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/denverdino/aliyungo/slb"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAliyunInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunInstanceCreate,
		Read:   resourceAliyunInstanceRead,
		Update: resourceAliyunInstanceUpdate,
		Delete: resourceAliyunInstanceDelete,

		Schema: map[string]*schema.Schema{
			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"image_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"instance_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"security_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"allocate_public_ip": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},

			"instance_name": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateInstanceName,
			},

			"description": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateInstanceDescription,
			},

			"instance_network_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateInstanceNetworkType,
			},

			"internet_charge_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateInternetChargeType,
			},
			"internet_max_bandwidth_in": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"internet_max_bandwidth_out": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateInternetMaxBandWidthOut,
			},
			"host_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"password": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"io_optimized": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateIoOptimized,
			},

			"system_disk_category": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "cloud",
				Optional: true,
				ForceNew: true,
			},
			"system_disk_size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			//subnet_id and vswitch_id both exists, cause compatible old version, and aws habit.
			"subnet_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"vswitch_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"instance_charge_type": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateInstanceChargeType,
			},
			"period": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"public_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"private_ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),

			"load_balancer": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},

			"load_balancer_weight": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
		},
	}
}

func resourceAliyunInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	args, err := buildAliyunInstanceArgs(d, meta)
	if err != nil {
		return err
	}

	instanceID, err := conn.CreateInstance(args)
	if err != nil {
		return fmt.Errorf("Error creating Aliyun ecs instance: %s", err)
	}

	d.SetId(instanceID)

	d.Partial(true)
	d.SetPartial("security_group_id")
	d.SetPartial("instance_name")
	d.SetPartial("description")
	d.SetPartial("password")
	if d.Get("subnet_id") != "" || d.Get("vswitch_id") != "" {
		d.SetPartial("subnet_id")
		d.SetPartial("vswitch_id")
	}
	d.SetPartial("system_disk_category")
	d.SetPartial("instance_charge_type")
	d.SetPartial("internet_charge_type")
	d.SetPartial("availability_zone")
	d.SetPartial("allocate_public_ip")

	if d.Get("allocate_public_ip").(bool) {
		ipAddress, err := conn.AllocatePublicIpAddress(d.Id())
		if err != nil {
			log.Printf("[DEBUG] AllocatePublicIpAddress for instance got error: %s", err)
		} else {
			d.Set("public_ip", ipAddress)
		}
	}

	// after instance created, its status is pending,
	// so we need to wait it become to stopped and then start it
	if err := conn.WaitForInstance(d.Id(), ecs.Stopped, defaultTimeout); err != nil {
		log.Printf("[DEBUG] WaitForInstance %s got error: %s", ecs.Stopped, err)
	}

	if err := conn.StartInstance(d.Id()); err != nil {
		return fmt.Errorf("Start instance got error: %s", err)
	}

	if err := conn.WaitForInstance(d.Id(), ecs.Running, defaultTimeout); err != nil {
		log.Printf("[DEBUG] WaitForInstance %s got error: %s", ecs.Running, err)
	}

	return resourceAliyunInstanceUpdate(d, meta)
}

func resourceAliyunInstanceRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AliyunClient).ecsconn

	instance, err := conn.DescribeInstanceAttribute(d.Id())
	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error DescribeInstanceAttribute: %s", err)
	}

	log.Printf("[DEBUG] DescribeInstanceAttribute for instance: %v", instance)

	d.Set("instance_name", instance.InstanceName)
	d.Set("description", instance.Description)
	d.Set("status", instance.Status)
	d.Set("availability_zone", instance.ZoneId)

	d.Set("image_id", instance.ImageId)
	d.Set("instance_type", instance.InstanceType)
	d.Set("internet_charge_type", instance.InternetChargeType)
	d.Set("io_optimized", instance.IoOptimized)

	d.Set("host_name", instance.HostName)

	// private ip only support vpc instance
	if d.Get("instance_network_type") == VpcNet {
		d.Set("private_ip", instance.VpcAttributes.PrivateIpAddress.IpAddress[0])
		d.Set("subnet_id", instance.VpcAttributes.VSwitchId)
		d.Set("vswitch_id", instance.VpcAttributes.VSwitchId)
	} else {
		d.Set("private_ip", instance.InnerIpAddress)
	}

	tags, _, err := conn.DescribeTags(&ecs.DescribeTagsArgs{
		RegionId:     getRegion(d, meta),
		ResourceType: ecs.TagResourceInstance,
		ResourceId:   d.Id(),
	})

	if err != nil {
		log.Printf("[DEBUG] DescribeTags for instance got error: %s", err)
	}

	log.Printf("[DEBUG] set tags")
	d.Set("tags", tagsToMap(tags))

	return nil
}

func resourceAliyunInstanceUpdate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*AliyunClient)
	conn := client.ecsconn
	slbconn := client.slbconn

	d.Partial(true)

	if err := setTags(client, ecs.TagResourceInstance, d); err != nil {
		log.Printf("[DEBUG] Set tags for instance got error: %s", err)
		return fmt.Errorf("Set tags for instance got error: %s", err)
	} else {
		d.SetPartial("tags")
	}

	if d.HasChange("password") {
		log.Printf("[DEBUG] ModifyInstanceAttribute password")
		val := d.Get("password").(string)
		args := &ecs.ModifyInstanceAttributeArgs{
			InstanceId: d.Id(),
			Password:   val,
		}

		if err := conn.ModifyInstanceAttribute(args); err != nil {
			return fmt.Errorf("Instance change password got error: %s", err)
		}

		if v, ok := d.GetOk("status"); ok && v.(string) != "" {
			if ecs.InstanceStatus(d.Get("status").(string)) == ecs.Running {
				log.Printf("[DEBUG] RebootInstance after change password")
				if err := conn.RebootInstance(d.Id(), false); err != nil {
					return fmt.Errorf("RebootInstance got error: %s", err)
				}

				if err := conn.WaitForInstance(d.Id(), ecs.Running, defaultTimeout); err != nil {
					return fmt.Errorf("WaitForInstance got error: %s", err)
				}
			}
		}

		d.SetPartial("password")
	}

	if d.HasChange("instance_name") {
		log.Printf("[DEBUG] ModifyInstanceAttribute instance_name")
		val := d.Get("instance_name").(string)
		args := &ecs.ModifyInstanceAttributeArgs{
			InstanceId:   d.Id(),
			InstanceName: val,
		}

		if err := conn.ModifyInstanceAttribute(args); err != nil {
			return fmt.Errorf("Modify instance name got error: %s", err)
		}

		d.SetPartial("instance_name")
	}

	if d.HasChange("description") {
		log.Printf("[DEBUG] ModifyInstanceAttribute description")
		val := d.Get("description").(string)
		args := &ecs.ModifyInstanceAttributeArgs{
			InstanceId:  d.Id(),
			Description: val,
		}

		if err := conn.ModifyInstanceAttribute(args); err != nil {
			return fmt.Errorf("Modify instance description got error: %s", err)
		}

		d.SetPartial("description")
	}

	if d.HasChange("host_name") {
		log.Printf("[DEBUG] ModifyInstanceAttribute host_name")
		val := d.Get("host_name").(string)
		args := &ecs.ModifyInstanceAttributeArgs{
			InstanceId: d.Id(),
			HostName:   val,
		}

		if err := conn.ModifyInstanceAttribute(args); err != nil {
			return fmt.Errorf("Modify instance host_name got error: %s", err)
		}

		d.SetPartial("host_name")
	}

	if d.HasChange("load_balancer") || d.HasChange("load_balancer_weight") {
		log.Printf("[DEBUG] ModifyInstanceAttribute load_balancer")
		loadBalanderId := d.Get("load_balancer").(string)

		var weight int = 100
		if v, ok := d.GetOk("load_balancer_weight"); ok {
			weight = v.(int)
		}

		log.Printf("[DEBUG] load_balancer weight is %s", weight)

		addBackendServerList := complexBackendServer(d.Id(), weight)
		removeBackendServerList := complexBackendServer(d.Id(), weight)

		if len(removeBackendServerList) > 0 {
			removeBackendServers := make([]string, 0, 1)
			removeBackendServers = append(removeBackendServers, d.Id())
			_, err := slbconn.RemoveBackendServers(loadBalanderId, removeBackendServers)
			if err != nil {
				return fmt.Errorf("RemoveBackendServers got error: %s", err)
			}
		}

		if len(addBackendServerList) > 0 {
			_, err := slbconn.AddBackendServers(loadBalanderId, addBackendServerList)
			if err != nil {
				return fmt.Errorf("AddBackendServers got error: %s", err)
			}
		}

		d.SetPartial("load_balancer")
	}

	d.Partial(false)

	return resourceAliyunInstanceRead(d, meta)
}

func complexBackendServer(instanceId string, weight int) []slb.BackendServerType {
	result := make([]slb.BackendServerType, 0, 1)
	backendServer := slb.BackendServerType{
		ServerId: instanceId,
		Weight:   weight,
	}
	result = append(result, backendServer)
	return result
}

func resourceAliyunInstanceDelete(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AliyunClient).ecsconn

	instance, err := conn.DescribeInstanceAttribute(d.Id())
	if err != nil {
		if notFoundError(err) {
			return nil
		}
		return fmt.Errorf("Error DescribeInstanceAttribute: %s", err)
	}

	if instance.Status != ecs.Stopped {
		if err := conn.StopInstance(d.Id(), true); err != nil {
			return err
		}

		if err := conn.WaitForInstance(d.Id(), ecs.Stopped, defaultTimeout); err != nil {
			return err
		}
	}

	if err := conn.DeleteInstance(d.Id()); err != nil {
		return err
	}

	return nil
}

func buildAliyunInstanceArgs(d *schema.ResourceData, meta interface{}) (*ecs.CreateInstanceArgs, error) {

	client := meta.(*AliyunClient)

	args := &ecs.CreateInstanceArgs{
		RegionId:         getRegion(d, meta),
		InstanceType:     d.Get("instance_type").(string),
		SecurityGroupId:  d.Get("security_group_id").(string),
		PrivateIpAddress: d.Get("private_ip").(string),
	}

	imageID := d.Get("image_id").(string)
	if _, err := client.DescribeImage(imageID); err != nil {
		return nil, err
	}

	args.ImageId = imageID

	zoneID := d.Get("availability_zone").(string)

	zone, err := client.DescribeZone(zoneID)
	if err != nil {
		return nil, err
	}

	if err := client.ResourceAvailable(zone, ecs.ResourceTypeInstance); err != nil {
		return nil, err
	}

	args.ZoneId = zoneID

	systemDiskCategory := ecs.DiskCategory(d.Get("system_disk_category").(string))

	if err := client.DiskAvailable(zone, systemDiskCategory); err != nil {
		return nil, err
	}

	args.SystemDisk = ecs.SystemDiskType{
		Category: systemDiskCategory,
	}

	if v := d.Get("instance_name").(string); v != "" {
		args.InstanceName = v
	}

	if v := d.Get("description").(string); v != "" {
		args.Description = v
	}

	log.Printf("[DEBUG] internet_charge_type is %s", d.Get("internet_charge_type").(string))
	if v := d.Get("internet_charge_type").(string); v != "" {
		args.InternetChargeType = common.InternetChargeType(v)
	}

	if v := d.Get("internet_max_bandwidth_out").(int); v != 0 {
		args.InternetMaxBandwidthOut = v
	}

	if v := d.Get("host_name").(string); v != "" {
		args.HostName = v
	}

	if v := d.Get("password").(string); v != "" {
		args.Password = v
	}

	if v := d.Get("io_optimized").(string); v != "" {
		args.IoOptimized = ecs.IoOptimized(v)
	}

	vswitchValue := d.Get("subnet_id").(string)
	if vswitchValue == "" {
		vswitchValue = d.Get("vswitch_id").(string)
	}
	if vswitchValue != "" {
		args.VSwitchId = vswitchValue
	}

	if v := d.Get("instance_charge_type").(string); v != "" {
		args.InstanceChargeType = common.InstanceChargeType(v)
	}

	log.Printf("[DEBUG] period is %s", d.Get("period").(int))
	if v := d.Get("period").(int); v != 0 {
		args.Period = v
	} else if args.InstanceChargeType == common.PrePaid {
		return nil, fmt.Errorf("period is required for instance_charge_type is PrePaid")
	}

	return args, nil
}
