package aws

import (
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/inspector"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAWSInspectorAssessmentTarget() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsInspectorAssessmentTargetCreate,
		Read:   resourceAwsInspectorAssessmentTargetRead,
		Update: resourceAwsInspectorAssessmentTargetUpdate,
		Delete: resourceAwsInspectorAssessmentTargetDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"resource_group_arn": {
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

func resourceAwsInspectorAssessmentTargetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	targetName := d.Get("name").(string)
	resourceGroupArn := d.Get("resource_group_arn").(string)

	resp, err := conn.CreateAssessmentTarget(&inspector.CreateAssessmentTargetInput{
		AssessmentTargetName: aws.String(targetName),
		ResourceGroupArn:     aws.String(resourceGroupArn),
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Inspector Assessment %s created", *resp.AssessmentTargetArn)

	d.Set("arn", resp.AssessmentTargetArn)
	d.SetId(*resp.AssessmentTargetArn)

	return resourceAwsInspectorAssessmentTargetRead(d, meta)
}

func resourceAwsInspectorAssessmentTargetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	resp, err := conn.DescribeAssessmentTargets(&inspector.DescribeAssessmentTargetsInput{
		AssessmentTargetArns: []*string{
			aws.String(d.Id()),
		},
	})

	if err != nil {
		if inspectorerr, ok := err.(awserr.Error); ok && inspectorerr.Code() == "InvalidInputException" {
			return nil
		} else {
			log.Printf("[ERROR] Error finding Inspector Assessment Target: %s", err)
			return err
		}
	}

	if resp.AssessmentTargets != nil && len(resp.AssessmentTargets) > 0 {
		d.Set("name", resp.AssessmentTargets[0].Name)
	}

	return nil
}

func resourceAwsInspectorAssessmentTargetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	input := inspector.UpdateAssessmentTargetInput{
		AssessmentTargetArn:  aws.String(d.Id()),
		AssessmentTargetName: aws.String(d.Get("name").(string)),
		ResourceGroupArn:     aws.String(d.Get("resource_group_arn").(string)),
	}

	_, err := conn.UpdateAssessmentTarget(&input)
	if err != nil {
		return err
	}

	log.Println("[DEBUG] Inspector Assessment Target updated")

	return resourceAwsInspectorAssessmentTargetRead(d, meta)
}

func resourceAwsInspectorAssessmentTargetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	return resource.Retry(60*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteAssessmentTarget(&inspector.DeleteAssessmentTargetInput{
			AssessmentTargetArn: aws.String(d.Id()),
		})
		if err != nil {
			if inspectorerr, ok := err.(awserr.Error); ok && inspectorerr.Code() == "AssessmentRunInProgressException" {
				log.Printf("[ERROR] Assement Run in progress: %s", err)
				return resource.RetryableError(err)
			} else {
				log.Printf("[ERROR] Error deleting Assement Target: %s", err)
				return resource.NonRetryableError(err)
			}
		}
		return nil
	})

}
