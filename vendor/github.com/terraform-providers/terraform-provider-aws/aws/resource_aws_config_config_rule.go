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
	"github.com/aws/aws-sdk-go/service/configservice"
)

func resourceAwsConfigConfigRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsConfigConfigRulePut,
		Read:   resourceAwsConfigConfigRuleRead,
		Update: resourceAwsConfigConfigRulePut,
		Delete: resourceAwsConfigConfigRuleDelete,

		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringLenBetween(0, 64),
			},
			"rule_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringLenBetween(0, 256),
			},
			"input_parameters": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateJsonString,
			},
			"maximum_execution_frequency": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateConfigExecutionFrequency(),
			},
			"scope": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"compliance_resource_id": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringLenBetween(0, 256),
						},
						"compliance_resource_types": {
							Type:     schema.TypeSet,
							Optional: true,
							MaxItems: 100,
							Elem: &schema.Schema{
								Type:         schema.TypeString,
								ValidateFunc: validation.StringLenBetween(0, 256),
							},
							Set: schema.HashString,
						},
						"tag_key": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringLenBetween(0, 128),
						},
						"tag_value": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringLenBetween(0, 256),
						},
					},
				},
			},
			"source": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Required: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"owner": {
							Type:     schema.TypeString,
							Required: true,
							ValidateFunc: validation.StringInSlice([]string{
								configservice.OwnerCustomLambda,
								configservice.OwnerAws,
							}, false),
						},
						"source_detail": {
							Type:     schema.TypeSet,
							Set:      configRuleSourceDetailsHash,
							Optional: true,
							MaxItems: 25,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"event_source": {
										Type:     schema.TypeString,
										Optional: true,
										Default:  "aws.config",
									},
									"maximum_execution_frequency": {
										Type:         schema.TypeString,
										Optional:     true,
										ValidateFunc: validateConfigExecutionFrequency(),
									},
									"message_type": {
										Type:     schema.TypeString,
										Optional: true,
									},
								},
							},
						},
						"source_identifier": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(0, 256),
						},
					},
				},
			},
		},
	}
}

func resourceAwsConfigConfigRulePut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	name := d.Get("name").(string)
	ruleInput := configservice.ConfigRule{
		ConfigRuleName: aws.String(name),
		Scope:          expandConfigRuleScope(d.Get("scope").([]interface{})),
		Source:         expandConfigRuleSource(d.Get("source").([]interface{})),
	}

	if v, ok := d.GetOk("description"); ok {
		ruleInput.Description = aws.String(v.(string))
	}
	if v, ok := d.GetOk("input_parameters"); ok {
		ruleInput.InputParameters = aws.String(v.(string))
	}
	if v, ok := d.GetOk("maximum_execution_frequency"); ok {
		ruleInput.MaximumExecutionFrequency = aws.String(v.(string))
	}

	input := configservice.PutConfigRuleInput{
		ConfigRule: &ruleInput,
	}
	log.Printf("[DEBUG] Creating AWSConfig config rule: %s", input)
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		_, err := conn.PutConfigRule(&input)
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if awsErr.Code() == "InsufficientPermissionsException" {
					// IAM is eventually consistent
					return resource.RetryableError(err)
				}
			}

			return resource.NonRetryableError(fmt.Errorf("Failed to create AWSConfig rule: %s", err))
		}

		return nil
	})
	if err != nil {
		return err
	}

	d.SetId(name)

	log.Printf("[DEBUG] AWSConfig config rule %q created", name)

	return resourceAwsConfigConfigRuleRead(d, meta)
}

func resourceAwsConfigConfigRuleRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	out, err := conn.DescribeConfigRules(&configservice.DescribeConfigRulesInput{
		ConfigRuleNames: []*string{aws.String(d.Id())},
	})
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoSuchConfigRuleException" {
			log.Printf("[WARN] Config Rule %q is gone (NoSuchConfigRuleException)", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}

	numberOfRules := len(out.ConfigRules)
	if numberOfRules < 1 {
		log.Printf("[WARN] Config Rule %q is gone (no rules found)", d.Id())
		d.SetId("")
		return nil
	}

	if numberOfRules > 1 {
		return fmt.Errorf("Expected exactly 1 Config Rule, received %d: %#v",
			numberOfRules, out.ConfigRules)
	}

	log.Printf("[DEBUG] AWS Config config rule received: %s", out)

	rule := out.ConfigRules[0]
	d.Set("arn", rule.ConfigRuleArn)
	d.Set("rule_id", rule.ConfigRuleId)
	d.Set("name", rule.ConfigRuleName)
	d.Set("description", rule.Description)
	d.Set("input_parameters", rule.InputParameters)
	d.Set("maximum_execution_frequency", rule.MaximumExecutionFrequency)

	if rule.Scope != nil {
		d.Set("scope", flattenConfigRuleScope(rule.Scope))
	}

	d.Set("source", flattenConfigRuleSource(rule.Source))

	return nil
}

func resourceAwsConfigConfigRuleDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	name := d.Get("name").(string)

	log.Printf("[DEBUG] Deleting AWS Config config rule %q", name)
	err := resource.Retry(2*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteConfigRule(&configservice.DeleteConfigRuleInput{
			ConfigRuleName: aws.String(name),
		})
		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "ResourceInUseException" {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("Deleting Config Rule failed: %s", err)
	}

	conf := resource.StateChangeConf{
		Pending: []string{
			configservice.ConfigRuleStateActive,
			configservice.ConfigRuleStateDeleting,
			configservice.ConfigRuleStateDeletingResults,
			configservice.ConfigRuleStateEvaluating,
		},
		Target:  []string{""},
		Timeout: 5 * time.Minute,
		Refresh: func() (interface{}, string, error) {
			out, err := conn.DescribeConfigRules(&configservice.DescribeConfigRulesInput{
				ConfigRuleNames: []*string{aws.String(d.Id())},
			})
			if err != nil {
				if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoSuchConfigRuleException" {
					return 42, "", nil
				}
				return 42, "", fmt.Errorf("Failed to describe config rule %q: %s", d.Id(), err)
			}
			if len(out.ConfigRules) < 1 {
				return 42, "", nil
			}
			rule := out.ConfigRules[0]
			return out, *rule.ConfigRuleState, nil
		},
	}
	_, err = conf.WaitForState()
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] AWS Config config rule %q deleted", name)

	return nil
}

func configRuleSourceDetailsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})
	if v, ok := m["message_type"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["event_source"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	if v, ok := m["maximum_execution_frequency"]; ok {
		buf.WriteString(fmt.Sprintf("%s-", v.(string)))
	}
	return hashcode.String(buf.String())
}
