package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

var ssmPatchComplianceLevels = []string{
	ssm.PatchComplianceLevelCritical,
	ssm.PatchComplianceLevelHigh,
	ssm.PatchComplianceLevelMedium,
	ssm.PatchComplianceLevelLow,
	ssm.PatchComplianceLevelInformational,
	ssm.PatchComplianceLevelUnspecified,
}

var ssmPatchOSs = []string{
	ssm.OperatingSystemWindows,
	ssm.OperatingSystemAmazonLinux,
	ssm.OperatingSystemUbuntu,
	ssm.OperatingSystemRedhatEnterpriseLinux,
}

func resourceAwsSsmPatchBaseline() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSsmPatchBaselineCreate,
		Read:   resourceAwsSsmPatchBaselineRead,
		Update: resourceAwsSsmPatchBaselineUpdate,
		Delete: resourceAwsSsmPatchBaselineDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
			},

			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"global_filter": {
				Type:     schema.TypeList,
				Optional: true,
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
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"approve_after_days": {
							Type:     schema.TypeInt,
							Required: true,
						},

						"compliance_level": {
							Type:         schema.TypeString,
							Optional:     true,
							Default:      ssm.PatchComplianceLevelUnspecified,
							ValidateFunc: validation.StringInSlice(ssmPatchComplianceLevels, false),
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
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"rejected_patches": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"operating_system": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      ssm.OperatingSystemWindows,
				ValidateFunc: validation.StringInSlice(ssmPatchOSs, false),
			},

			"approved_patches_compliance_level": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      ssm.PatchComplianceLevelUnspecified,
				ValidateFunc: validation.StringInSlice(ssmPatchComplianceLevels, false),
			},
		},
	}
}

func resourceAwsSsmPatchBaselineCreate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	params := &ssm.CreatePatchBaselineInput{
		Name: aws.String(d.Get("name").(string)),
		ApprovedPatchesComplianceLevel: aws.String(d.Get("approved_patches_compliance_level").(string)),
		OperatingSystem:                aws.String(d.Get("operating_system").(string)),
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

func resourceAwsSsmPatchBaselineUpdate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	params := &ssm.UpdatePatchBaselineInput{
		BaselineId: aws.String(d.Id()),
	}

	if d.HasChange("name") {
		params.Name = aws.String(d.Get("name").(string))
	}

	if d.HasChange("description") {
		params.Description = aws.String(d.Get("description").(string))
	}

	if d.HasChange("approved_patches") {
		params.ApprovedPatches = expandStringList(d.Get("approved_patches").(*schema.Set).List())
	}

	if d.HasChange("rejected_patches") {
		params.RejectedPatches = expandStringList(d.Get("rejected_patches").(*schema.Set).List())
	}

	if d.HasChange("approved_patches_compliance_level") {
		params.ApprovedPatchesComplianceLevel = aws.String(d.Get("approved_patches_compliance_level").(string))
	}

	if d.HasChange("approval_rule") {
		params.ApprovalRules = expandAwsSsmPatchRuleGroup(d)
	}

	if d.HasChange("global_filter") {
		params.GlobalFilters = expandAwsSsmPatchFilterGroup(d)
	}

	_, err := ssmconn.UpdatePatchBaseline(params)
	if err != nil {
		return err
	}

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
	d.Set("operating_system", resp.OperatingSystem)
	d.Set("approved_patches_compliance_level", resp.ApprovedPatchesComplianceLevel)
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
			ComplianceLevel:  aws.String(rCfg["compliance_level"].(string)),
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
		r["compliance_level"] = *rule.ComplianceLevel
		r["patch_filter"] = flattenAwsSsmPatchFilterGroup(rule.PatchFilterGroup)
		result = append(result, r)
	}

	return result
}
