package alicloud

import (
	"fmt"
	"github.com/denverdino/aliyungo/common"
	"github.com/denverdino/aliyungo/ess"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"strings"
	"time"
)

func resourceAlicloudEssScalingGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunEssScalingGroupCreate,
		Read:   resourceAliyunEssScalingGroupRead,
		Update: resourceAliyunEssScalingGroupUpdate,
		Delete: resourceAliyunEssScalingGroupDelete,

		Schema: map[string]*schema.Schema{
			"min_size": &schema.Schema{
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validateIntegerInRange(0, 100),
			},
			"max_size": &schema.Schema{
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validateIntegerInRange(0, 100),
			},
			"scaling_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"default_cooldown": &schema.Schema{
				Type:         schema.TypeInt,
				Default:      300,
				Optional:     true,
				ValidateFunc: validateIntegerInRange(0, 86400),
			},
			"vswitch_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"removal_policies": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				MaxItems: 2,
			},
			"db_instance_ids": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				MaxItems: 3,
			},
			"loadbalancer_ids": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
		},
	}
}

func resourceAliyunEssScalingGroupCreate(d *schema.ResourceData, meta interface{}) error {

	args, err := buildAlicloudEssScalingGroupArgs(d, meta)
	if err != nil {
		return err
	}

	essconn := meta.(*AliyunClient).essconn

	scaling, err := essconn.CreateScalingGroup(args)
	if err != nil {
		return err
	}

	d.SetId(scaling.ScalingGroupId)

	return resourceAliyunEssScalingGroupUpdate(d, meta)
}

func resourceAliyunEssScalingGroupRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*AliyunClient)

	scaling, err := client.DescribeScalingGroupById(d.Id())
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == InstanceNotfound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error Describe ESS scaling group Attribute: %#v", err)
	}

	d.Set("min_size", scaling.MinSize)
	d.Set("max_size", scaling.MaxSize)
	d.Set("scaling_group_name", scaling.ScalingGroupName)
	d.Set("default_cooldown", scaling.DefaultCooldown)
	d.Set("removal_policies", scaling.RemovalPolicies)
	d.Set("db_instance_ids", scaling.DBInstanceIds)
	d.Set("loadbalancer_ids", scaling.LoadBalancerId)

	return nil
}

func resourceAliyunEssScalingGroupUpdate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AliyunClient).essconn
	args := &ess.ModifyScalingGroupArgs{
		ScalingGroupId: d.Id(),
	}

	if d.HasChange("scaling_group_name") {
		args.ScalingGroupName = d.Get("scaling_group_name").(string)
	}

	if d.HasChange("min_size") {
		args.MinSize = d.Get("min_size").(int)
	}

	if d.HasChange("max_size") {
		args.MaxSize = d.Get("max_size").(int)
	}

	if d.HasChange("default_cooldown") {
		args.DefaultCooldown = d.Get("default_cooldown").(int)
	}

	if d.HasChange("removal_policies") {
		policyStrings := d.Get("removal_policies").([]interface{})
		args.RemovalPolicy = expandStringList(policyStrings)
	}

	if _, err := conn.ModifyScalingGroup(args); err != nil {
		return err
	}

	return resourceAliyunEssScalingGroupRead(d, meta)
}

func resourceAliyunEssScalingGroupDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)

	return resource.Retry(2*time.Minute, func() *resource.RetryError {
		err := client.DeleteScalingGroupById(d.Id())

		if err != nil {
			e, _ := err.(*common.Error)
			if e.ErrorResponse.Code != InvalidScalingGroupIdNotFound {
				return resource.RetryableError(fmt.Errorf("Scaling group in use - trying again while it is deleted."))
			}
		}

		_, err = client.DescribeScalingGroupById(d.Id())
		if err != nil {
			if notFoundError(err) {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(fmt.Errorf("Scaling group in use - trying again while it is deleted."))
	})
}

func buildAlicloudEssScalingGroupArgs(d *schema.ResourceData, meta interface{}) (*ess.CreateScalingGroupArgs, error) {
	client := meta.(*AliyunClient)
	args := &ess.CreateScalingGroupArgs{
		RegionId:        getRegion(d, meta),
		MinSize:         d.Get("min_size").(int),
		MaxSize:         d.Get("max_size").(int),
		DefaultCooldown: d.Get("default_cooldown").(int),
	}

	if v := d.Get("scaling_group_name").(string); v != "" {
		args.ScalingGroupName = v
	}

	if v := d.Get("vswitch_id").(string); v != "" {
		args.VSwitchId = v

		// get vpcId
		vpcId, err := client.GetVpcIdByVSwitchId(v)

		if err != nil {
			return nil, fmt.Errorf("VswitchId %s is not valid of current region", v)
		}
		// fill vpcId by vswitchId
		args.VpcId = vpcId

	}

	dbs, ok := d.GetOk("db_instance_ids")
	if ok {
		dbsStrings := dbs.([]interface{})
		args.DBInstanceId = expandStringList(dbsStrings)
	}

	lbs, ok := d.GetOk("loadbalancer_ids")
	if ok {
		lbsStrings := lbs.([]interface{})
		args.LoadBalancerId = strings.Join(expandStringList(lbsStrings), COMMA_SEPARATED)
	}

	return args, nil
}
