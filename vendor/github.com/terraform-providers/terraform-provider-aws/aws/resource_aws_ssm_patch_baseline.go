package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSsmPatchBaseline() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSsmPatchBaselineCreate,
		Read:   resourceAwsSsmPatchBaselineRead,
		Delete: resourceAwsSsmPatchBaselineDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"global_filter": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 4,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
						},
						"values": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"approval_rule": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"approve_after_days": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"patch_filter": {
							Type:     schema.TypeList,
							Required: true,
							MaxItems: 10,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:     schema.TypeString,
										Required: true,
									},
									"values": {
										Type:     schema.TypeList,
										Required: true,
										Elem:     &schema.Schema{Type: schema.TypeString},
									},
								},
							},
						},
					},
				},
			},

			"approved_patches": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"rejected_patches": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceAwsSsmPatchBaselineCreate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	params := &ssm.CreatePatchBaselineInput{
		Name: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("description"); ok {
		params.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("approved_patches"); ok && v.(*schema.Set).Len() > 0 {
		params.ApprovedPatches = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("rejected_patches"); ok && v.(*schema.Set).Len() > 0 {
		params.RejectedPatches = expandStringList(v.(*schema.Set).List())
	}

	if _, ok := d.GetOk("global_filter"); ok {
		params.GlobalFilters = expandAwsSsmPatchFilterGroup(d)
	}

	if _, ok := d.GetOk("approval_rule"); ok {
		params.ApprovalRules = expandAwsSsmPatchRuleGroup(d)
	}

	resp, err := ssmconn.CreatePatchBaseline(params)
	if err != nil {
		return err
	}

	d.SetId(*resp.BaselineId)
	return resourceAwsSsmPatchBaselineRead(d, meta)
}

func resourceAwsSsmPatchBaselineRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	params := &ssm.GetPatchBaselineInput{
		BaselineId: aws.String(d.Id()),
	}

	resp, err := ssmconn.GetPatchBaseline(params)
	if err != nil {
		return err
	}

	d.Set("name", resp.Name)
	d.Set("description", resp.Description)
	d.Set("approved_patches", flattenStringList(resp.ApprovedPatches))
	d.Set("rejected_patches", flattenStringList(resp.RejectedPatches))

	if err := d.Set("global_filter", flattenAwsSsmPatchFilterGroup(resp.GlobalFilters)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting global filters error: %#v", err)
	}

	if err := d.Set("approval_rule", flattenAwsSsmPatchRuleGroup(resp.ApprovalRules)); err != nil {
		return fmt.Errorf("[DEBUG] Error setting approval rules error: %#v", err)
	}

	return nil
}

func resourceAwsSsmPatchBaselineDelete(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Deleting SSM Patch Baseline: %s", d.Id())

	params := &ssm.DeletePatchBaselineInput{
		BaselineId: aws.String(d.Id()),
	}

	_, err := ssmconn.DeletePatchBaseline(params)
	if err != nil {
		return err
	}

	return nil
}

func expandAwsSsmPatchFilterGroup(d *schema.ResourceData) *ssm.PatchFilterGroup {
	var filters []*ssm.PatchFilter

	filterConfig := d.Get("global_filter").([]interface{})

	for _, fConfig := range filterConfig {
		config := fConfig.(map[string]interface{})

		filter := &ssm.PatchFilter{
			Key:    aws.String(config["key"].(string)),
			Values: expandStringList(config["values"].([]interface{})),
		}

		filters = append(filters, filter)
	}

	return &ssm.PatchFilterGroup{
		PatchFilters: filters,
	}
}

func flattenAwsSsmPatchFilterGroup(group *ssm.PatchFilterGroup) []map[string]interface{} {
	if len(group.PatchFilters) == 0 {
		return nil
	}

	result := make([]map[string]interface{}, 0, len(group.PatchFilters))

	for _, filter := range group.PatchFilters {
		f := make(map[string]interface{})
		f["key"] = *filter.Key
		f["values"] = flattenStringList(filter.Values)

		result = append(result, f)
	}

	return result
}

func expandAwsSsmPatchRuleGroup(d *schema.ResourceData) *ssm.PatchRuleGroup {
	var rules []*ssm.PatchRule

	ruleConfig := d.Get("approval_rule").([]interface{})

	for _, rConfig := range ruleConfig {
		rCfg := rConfig.(map[string]interface{})

		var filters []*ssm.PatchFilter
		filterConfig := rCfg["patch_filter"].([]interface{})

		for _, fConfig := range filterConfig {
			fCfg := fConfig.(map[string]interface{})

			filter := &ssm.PatchFilter{
				Key:    aws.String(fCfg["key"].(string)),
				Values: expandStringList(fCfg["values"].([]interface{})),
			}

			filters = append(filters, filter)
		}

		filterGroup := &ssm.PatchFilterGroup{
			PatchFilters: filters,
		}

		rule := &ssm.PatchRule{
			ApproveAfterDays: aws.Int64(int64(rCfg["approve_after_days"].(int))),
			PatchFilterGroup: filterGroup,
		}

		rules = append(rules, rule)
	}

	return &ssm.PatchRuleGroup{
		PatchRules: rules,
	}
}

func flattenAwsSsmPatchRuleGroup(group *ssm.PatchRuleGroup) []map[string]interface{} {
	if len(group.PatchRules) == 0 {
		return nil
	}

	result := make([]map[string]interface{}, 0, len(group.PatchRules))

	for _, rule := range group.PatchRules {
		r := make(map[string]interface{})
		r["approve_after_days"] = *rule.ApproveAfterDays
		r["patch_filter"] = flattenAwsSsmPatchFilterGroup(rule.PatchFilterGroup)
		result = append(result, r)
	}

	return result
}
