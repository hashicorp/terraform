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

func resourceAlicloudEssScalingRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAliyunEssScalingRuleCreate,
		Read:   resourceAliyunEssScalingRuleRead,
		Update: resourceAliyunEssScalingRuleUpdate,
		Delete: resourceAliyunEssScalingRuleDelete,

		Schema: map[string]*schema.Schema{
			"scaling_group_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"adjustment_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validateAllowedStringValue([]string{string(ess.QuantityChangeInCapacity),
					string(ess.PercentChangeInCapacity), string(ess.TotalCapacity)}),
			},
			"adjustment_value": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"scaling_rule_name": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
			},
			"ari": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"cooldown": &schema.Schema{
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validateIntegerInRange(0, 86400),
			},
		},
	}
}

func resourceAliyunEssScalingRuleCreate(d *schema.ResourceData, meta interface{}) error {

	args, err := buildAlicloudEssScalingRuleArgs(d, meta)
	if err != nil {
		return err
	}

	essconn := meta.(*AliyunClient).essconn

	rule, err := essconn.CreateScalingRule(args)
	if err != nil {
		return err
	}

	d.SetId(d.Get("scaling_group_id").(string) + COLON_SEPARATED + rule.ScalingRuleId)

	return resourceAliyunEssScalingRuleUpdate(d, meta)
}

func resourceAliyunEssScalingRuleRead(d *schema.ResourceData, meta interface{}) error {

	client := meta.(*AliyunClient)
	ids := strings.Split(d.Id(), COLON_SEPARATED)

	rule, err := client.DescribeScalingRuleById(ids[0], ids[1])
	if err != nil {
		if e, ok := err.(*common.Error); ok && e.Code == InstanceNotfound {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error Describe ESS scaling rule Attribute: %#v", err)
	}

	d.Set("scaling_group_id", rule.ScalingGroupId)
	d.Set("ari", rule.ScalingRuleAri)
	d.Set("adjustment_type", rule.AdjustmentType)
	d.Set("adjustment_value", rule.AdjustmentValue)
	d.Set("scaling_rule_name", rule.ScalingRuleName)
	d.Set("cooldown", rule.Cooldown)

	return nil
}

func resourceAliyunEssScalingRuleDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*AliyunClient)
	ids := strings.Split(d.Id(), COLON_SEPARATED)

	return resource.Retry(2*time.Minute, func() *resource.RetryError {
		err := client.DeleteScalingRuleById(ids[1])

		if err != nil {
			return resource.RetryableError(fmt.Errorf("Scaling rule in use - trying again while it is deleted."))
		}

		_, err = client.DescribeScalingRuleById(ids[0], ids[1])
		if err != nil {
			if notFoundError(err) {
				return nil
			}
			return resource.NonRetryableError(err)
		}

		return resource.RetryableError(fmt.Errorf("Scaling rule in use - trying again while it is deleted."))
	})
}

func resourceAliyunEssScalingRuleUpdate(d *schema.ResourceData, meta interface{}) error {

	conn := meta.(*AliyunClient).essconn
	ids := strings.Split(d.Id(), COLON_SEPARATED)

	args := &ess.ModifyScalingRuleArgs{
		ScalingRuleId: ids[1],
	}

	if d.HasChange("adjustment_type") {
		args.AdjustmentType = ess.AdjustmentType(d.Get("adjustment_type").(string))
	}

	if d.HasChange("adjustment_value") {
		args.AdjustmentValue = d.Get("adjustment_value").(int)
	}

	if d.HasChange("scaling_rule_name") {
		args.ScalingRuleName = d.Get("scaling_rule_name").(string)
	}

	if d.HasChange("cooldown") {
		args.Cooldown = d.Get("cooldown").(int)
	}

	if _, err := conn.ModifyScalingRule(args); err != nil {
		return err
	}

	return resourceAliyunEssScalingRuleRead(d, meta)
}

func buildAlicloudEssScalingRuleArgs(d *schema.ResourceData, meta interface{}) (*ess.CreateScalingRuleArgs, error) {
	args := &ess.CreateScalingRuleArgs{
		RegionId:        getRegion(d, meta),
		ScalingGroupId:  d.Get("scaling_group_id").(string),
		AdjustmentType:  ess.AdjustmentType(d.Get("adjustment_type").(string)),
		AdjustmentValue: d.Get("adjustment_value").(int),
	}

	if v := d.Get("scaling_rule_name").(string); v != "" {
		args.ScalingRuleName = v
	}

	if v := d.Get("cooldown").(int); v != 0 {
		args.Cooldown = v
	}

	return args, nil
}
