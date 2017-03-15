package alicloud

import (
	"fmt"
	"log"

	"encoding/base64"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ecs"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
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

			"security_groups": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},

			"allocate_public_ip": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			"instance_name": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "ECS-Instance",
				ValidateFunc: validateInstanceName,
			},

			"description": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateInstanceDescription,
			},

			"instance_network_type": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
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
				Type:      schema.TypeString,
				Optional:  true,
				Sensitive: true,
			},
			"io_optimized": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
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
				Computed: true, //add this schema cause subnet_id not used enter parameter, will different, so will be ForceNew
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

			"user_data": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"tags": tagsSchema(),
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
		return fmt.Errorf("Error creating Aliyun ecs instance: %#v", err)
	}

	d.SetId(instanceID)

	d.Set("password", d.Get("password"))
	d.Set("system_disk_category", d.Get("system_disk_category"))

	if d.Get("allocate_public_ip").(bool) {
		_, err := conn.AllocatePublicIpAddress(d.Id())
		if err != nil {
			log.Printf("[DEBUG] AllocatePublicIpAddress for instance got error: %#v", err)
		}
	}

	// after instance created, its status is pending,
	// so we need to wait it become to stopped and then start it
	if err := conn.WaitForInstance(d.Id(), ecs.Stopped, defaultTimeout); err != nil {
		log.Printf("[DEBUG] WaitForInstance %s got error: %#v", ecs.Stopped, err)
	}

	if err := conn.StartInstance(d.Id()); err != nil {
		return fmt.Errorf("Start instance got error: %#v", err)
	}

	if err := conn.WaitForInstance(d.Id(), ecs.Running, defaultTimeout); err != nil {
		log.Printf("[DEBUG] WaitForInstance %s got error: %#v", ecs.Running, err)
	}

	return resourceAliyunInstanceUpdate(d, meta)
}

func resourceAliyunInstanceRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	conn := client.ecsconn

	instance, err := client.QueryInstancesById(d.Id())
	if err != nil {
		if notFoundError(err) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error DescribeInstanceAttribute: %#v", err)
	}

	log.Printf("[DEBUG] DescribeInstanceAttribute for instance: %#v", instance)

	d.Set("instance_name", instance.InstanceName)
	d.Set("description", instance.Description)
	d.Set("status", instance.Status)
	d.Set("availability_zone", instance.ZoneId)
	d.Set("host_name", instance.HostName)
	d.Set("image_id", instance.ImageId)
	d.Set("instance_type", instance.InstanceType)

	// In Classic network, internet_charge_type is valid in any case, and its default value is 'PayByBanwidth'.
	// In VPC network, internet_charge_type is valid when instance has public ip, and its default value is 'PayByBanwidth'.
	d.Set("internet_charge_type", instance.InternetChargeType)

	if d.Get("allocate_public_ip").(bool) {
		d.Set("public_ip", instance.PublicIpAddress.IpAddress[0])
	}

	if ecs.StringOrBool(instance.IoOptimized).Value {
		d.Set("io_optimized", "optimized")
	} else {
		d.Set("io_optimized", "none")
	}

	log.Printf("instance.InternetChargeType: %#v", instance.InternetChargeType)

	d.Set("instance_network_type", instance.InstanceNetworkType)

	if d.Get("subnet_id").(string) != "" || d.Get("vswitch_id").(string) != "" {
		ipAddress := instance.VpcAttributes.PrivateIpAddress.IpAddress[0]
		d.Set("private_ip", ipAddress)
		d.Set("subnet_id", instance.VpcAttributes.VSwitchId)
		d.Set("vswitch_id", instance.VpcAttributes.VSwitchId)
	} else {
		ipAddress := strings.Join(ecs.IpAddressSetType(instance.InnerIpAddress).IpAddress, ",")
		d.Set("private_ip", ipAddress)
	}

	if d.Get("user_data").(string) != "" {
		ud, err := conn.DescribeUserdata(&ecs.DescribeUserdataArgs{
			RegionId:   getRegion(d, meta),
			InstanceId: d.Id(),
		})

		if err != nil {
			log.Printf("[ERROR] DescribeUserData for instance got error: %#v", err)
		}
		d.Set("user_data", userDataHashSum(ud.UserData))
	}

	tags, _, err := conn.DescribeTags(&ecs.DescribeTagsArgs{
		RegionId:     getRegion(d, meta),
		ResourceType: ecs.TagResourceInstance,
		ResourceId:   d.Id(),
	})

	if err != nil {
		log.Printf("[ERROR] DescribeTags for instance got error: %#v", err)
	}
	d.Set("tags", tagsToMap(tags))

	return nil
}

func resourceAliyunInstanceUpdate(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*AliyunClient)
	conn := client.ecsconn

	d.Partial(true)

	if err := setTags(client, ecs.TagResourceInstance, d); err != nil {
		log.Printf("[DEBUG] Set tags for instance got error: %#v", err)
		return fmt.Errorf("Set tags for instance got error: %#v", err)
	} else {
		d.SetPartial("tags")
	}

	attributeUpdate := false
	args := &ecs.ModifyInstanceAttributeArgs{
		InstanceId: d.Id(),
	}

	if d.HasChange("instance_name") {
		log.Printf("[DEBUG] ModifyInstanceAttribute instance_name")
		d.SetPartial("instance_name")
		args.InstanceName = d.Get("instance_name").(string)

		attributeUpdate = true
	}

	if d.HasChange("description") {
		log.Printf("[DEBUG] ModifyInstanceAttribute description")
		d.SetPartial("description")
		args.Description = d.Get("description").(string)

		attributeUpdate = true
	}

	if d.HasChange("host_name") {
		log.Printf("[DEBUG] ModifyInstanceAttribute host_name")
		d.SetPartial("host_name")
		args.HostName = d.Get("host_name").(string)

		attributeUpdate = true
	}

	passwordUpdate := false
	if d.HasChange("password") {
		log.Printf("[DEBUG] ModifyInstanceAttribute password")
		d.SetPartial("password")
		args.Password = d.Get("password").(string)

		attributeUpdate = true
		passwordUpdate = true
	}

	if attributeUpdate {
		if err := conn.ModifyInstanceAttribute(args); err != nil {
			return fmt.Errorf("Modify instance attribute got error: %#v", err)
		}
	}

	if passwordUpdate {
		if v, ok := d.GetOk("status"); ok && v.(string) != "" {
			if ecs.InstanceStatus(d.Get("status").(string)) == ecs.Running {
				log.Printf("[DEBUG] RebootInstance after change password")
				if err := conn.RebootInstance(d.Id(), false); err != nil {
					return fmt.Errorf("RebootInstance got error: %#v", err)
				}

				if err := conn.WaitForInstance(d.Id(), ecs.Running, defaultTimeout); err != nil {
					return fmt.Errorf("WaitForInstance got error: %#v", err)
				}
			}
		}
	}

	if d.HasChange("security_groups") {
		o, n := d.GetChange("security_groups")
		os := o.(*schema.Set)
		ns := n.(*schema.Set)

		rl := expandStringList(os.Difference(ns).List())
		al := expandStringList(ns.Difference(os).List())

		if len(al) > 0 {
			err := client.JoinSecurityGroups(d.Id(), al)
			if err != nil {
				return err
			}
		}
		if len(rl) > 0 {
			err := client.LeaveSecurityGroups(d.Id(), rl)
			if err != nil {
				return err
			}
		}

		d.SetPartial("security_groups")
	}

	d.Partial(false)
	return resourceAliyunInstanceRead(d, meta)
}

func resourceAliyunInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	conn := client.ecsconn

	instance, err := client.QueryInstancesById(d.Id())
	if err != nil {
		if notFoundError(err) {
			return nil
		}
		return fmt.Errorf("Error DescribeInstanceAttribute: %#v", err)
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
		PrivateIpAddress: d.Get("private_ip").(string),
	}

	imageID := d.Get("image_id").(string)

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

	sgs, ok := d.GetOk("security_groups")

	if ok {
		sgList := expandStringList(sgs.(*schema.Set).List())
		sg0 := sgList[0]
		// check security group instance exist
		_, err := client.DescribeSecurity(sg0)
		if err == nil {
			args.SecurityGroupId = sg0
		}

	}

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
		if d.Get("allocate_public_ip").(bool) && args.InternetMaxBandwidthOut <= 0 {
			return nil, fmt.Errorf("Invalid internet_max_bandwidth_out result in allocation public ip failed in the VPC.")
		}
	}

	if v := d.Get("instance_charge_type").(string); v != "" {
		args.InstanceChargeType = common.InstanceChargeType(v)
	}

	log.Printf("[DEBUG] period is %d", d.Get("period").(int))
	if v := d.Get("period").(int); v != 0 {
		args.Period = v
	} else if args.InstanceChargeType == common.PrePaid {
		return nil, fmt.Errorf("period is required for instance_charge_type is PrePaid")
	}

	if v := d.Get("user_data").(string); v != "" {
		args.UserData = v
	}

	return args, nil
}

func userDataHashSum(user_data string) string {
	// Check whether the user_data is not Base64 encoded.
	// Always calculate hash of base64 decoded value since we
	// check against double-encoding when setting it
	v, base64DecodeError := base64.StdEncoding.DecodeString(user_data)
	if base64DecodeError != nil {
		v = []byte(user_data)
	}
	return string(v)
}
