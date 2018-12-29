package aws

import (
	"fmt"
	"log"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func validateMonthlySpend(v interface{}, k string) (ws []string, errors []error) {
	vInt, _ := strconv.Atoi(v.(string))
	if vInt < 0 {
		errors = append(errors, fmt.Errorf("Error setting SMS preferences: monthly spend limit value [%d] must be >= 0!", vInt))
	}
	return
}

func validateDeliverySamplingRate(v interface{}, k string) (ws []string, errors []error) {
	vInt, _ := strconv.Atoi(v.(string))
	if vInt < 0 || vInt > 100 {
		errors = append(errors, fmt.Errorf("Error setting SMS preferences: default percentage of success to sample value [%d] must be between 0 and 100!", vInt))
	}
	return
}

func resourceAwsSnsSmsPreferences() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSnsSmsPreferencesSet,
		Read:   resourceAwsSnsSmsPreferencesGet,
		Update: resourceAwsSnsSmsPreferencesSet,
		Delete: resourceAwsSnsSmsPreferencesDelete,

		Schema: map[string]*schema.Schema{
			"monthly_spend_limit": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateMonthlySpend,
			},

			"delivery_status_iam_role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"delivery_status_success_sampling_rate": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validateDeliverySamplingRate,
			},

			"default_sender_id": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"default_sms_type": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"Promotional", "Transactional"}, false),
			},

			"usage_report_s3_bucket": {
				Type:     schema.TypeString,
				Optional: true,
			},
		},
	}
}

const resourceId = "aws_sns_sms_id"

var smsAttributeMap = map[string]string{
	"monthly_spend_limit":                   "MonthlySpendLimit",
	"delivery_status_iam_role_arn":          "DeliveryStatusIAMRole",
	"delivery_status_success_sampling_rate": "DeliveryStatusSuccessSamplingRate",
	"default_sender_id":                     "DefaultSenderID",
	"default_sms_type":                      "DefaultSMSType",
	"usage_report_s3_bucket":                "UsageReportS3Bucket",
}

var smsAttributeDefaultValues = map[string]string{
	"monthly_spend_limit":                   "",
	"delivery_status_iam_role_arn":          "",
	"delivery_status_success_sampling_rate": "",
	"default_sender_id":                     "",
	"default_sms_type":                      "",
	"usage_report_s3_bucket":                "",
}

func resourceAwsSnsSmsPreferencesSet(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	log.Printf("[DEBUG] SNS Set SMS preferences")

	monthlySpendLimit := d.Get("monthly_spend_limit").(string)
	deliveryStatusIamRoleArn := d.Get("delivery_status_iam_role_arn").(string)
	deliveryStatusSuccessSamplingRate := d.Get("delivery_status_success_sampling_rate").(string)
	defaultSenderId := d.Get("default_sender_id").(string)
	defaultSmsType := d.Get("default_sms_type").(string)
	usageReportS3Bucket := d.Get("usage_report_s3_bucket").(string)

	// Set preferences
	params := &sns.SetSMSAttributesInput{
		Attributes: map[string]*string{
			"MonthlySpendLimit":                 aws.String(monthlySpendLimit),
			"DeliveryStatusIAMRole":             aws.String(deliveryStatusIamRoleArn),
			"DeliveryStatusSuccessSamplingRate": aws.String(deliveryStatusSuccessSamplingRate),
			"DefaultSenderID":                   aws.String(defaultSenderId),
			"DefaultSMSType":                    aws.String(defaultSmsType),
			"UsageReportS3Bucket":               aws.String(usageReportS3Bucket),
		},
	}

	if _, err := snsconn.SetSMSAttributes(params); err != nil {
		return fmt.Errorf("Error setting SMS preferences: %s", err)
	}

	d.SetId(resourceId)
	return nil
}

func resourceAwsSnsSmsPreferencesGet(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	// Fetch ALL attributes
	attrs, err := snsconn.GetSMSAttributes(&sns.GetSMSAttributesInput{})
	if err != nil {
		return err
	}

	// Reset with default values first
	for tfAttrName, defValue := range smsAttributeDefaultValues {
		d.Set(tfAttrName, defValue)
	}

	// Apply existing settings
	if attrs.Attributes != nil && len(attrs.Attributes) > 0 {
		attrmap := attrs.Attributes
		for tfAttrName, snsAttrName := range smsAttributeMap {
			d.Set(tfAttrName, attrmap[snsAttrName])
		}
	}

	return nil
}

func resourceAwsSnsSmsPreferencesDelete(d *schema.ResourceData, meta interface{}) error {
	snsconn := meta.(*AWSClient).snsconn

	// Reset the attributes to their default value
	attrs := map[string]*string{}
	for tfAttrName, defValue := range smsAttributeDefaultValues {
		attrs[smsAttributeMap[tfAttrName]] = &defValue
	}

	params := &sns.SetSMSAttributesInput{Attributes: attrs}
	if _, err := snsconn.SetSMSAttributes(params); err != nil {
		return fmt.Errorf("Error resetting SMS preferences: %s", err)
	}

	return nil
}
