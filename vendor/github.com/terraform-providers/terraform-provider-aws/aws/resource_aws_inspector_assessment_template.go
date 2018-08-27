package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/inspector"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAWSInspectorAssessmentTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsInspectorAssessmentTemplateCreate,
		Read:   resourceAwsInspectorAssessmentTemplateRead,
		Delete: resourceAwsInspectorAssessmentTemplateDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"target_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"duration": {
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"rules_package_arns": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsInspectorAssessmentTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	rules := []*string{}
	if attr := d.Get("rules_package_arns").(*schema.Set); attr.Len() > 0 {
		rules = expandStringList(attr.List())
	}

	targetArn := d.Get("target_arn").(string)
	templateName := d.Get("name").(string)
	duration := int64(d.Get("duration").(int))

	resp, err := conn.CreateAssessmentTemplate(&inspector.CreateAssessmentTemplateInput{
		AssessmentTargetArn:    aws.String(targetArn),
		AssessmentTemplateName: aws.String(templateName),
		DurationInSeconds:      aws.Int64(duration),
		RulesPackageArns:       rules,
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Inspector Assessment Template %s created", *resp.AssessmentTemplateArn)

	d.Set("arn", resp.AssessmentTemplateArn)

	d.SetId(*resp.AssessmentTemplateArn)

	return resourceAwsInspectorAssessmentTemplateRead(d, meta)
}

func resourceAwsInspectorAssessmentTemplateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	resp, err := conn.DescribeAssessmentTemplates(&inspector.DescribeAssessmentTemplatesInput{
		AssessmentTemplateArns: []*string{
			aws.String(d.Id()),
		},
	},
	)
	if err != nil {
		if inspectorerr, ok := err.(awserr.Error); ok && inspectorerr.Code() == "InvalidInputException" {
			return nil
		} else {
			log.Printf("[ERROR] Error finding Inspector Assessment Template: %s", err)
			return err
		}
	}

	if resp.AssessmentTemplates != nil && len(resp.AssessmentTemplates) > 0 {
		d.Set("name", resp.AssessmentTemplates[0].Name)
	}
	return nil
}

func resourceAwsInspectorAssessmentTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	_, err := conn.DeleteAssessmentTemplate(&inspector.DeleteAssessmentTemplateInput{
		AssessmentTemplateArn: aws.String(d.Id()),
	})
	if err != nil {
		if inspectorerr, ok := err.(awserr.Error); ok && inspectorerr.Code() == "AssessmentRunInProgressException" {
			log.Printf("[ERROR] Assement Run in progress: %s", err)
			return err
		} else {
			log.Printf("[ERROR] Error deleting Assement Template: %s", err)
			return err
		}
	}

	return nil
}
