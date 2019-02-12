package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

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
				Optional: true,
			},
		},
	}
}

func resourceAwsInspectorAssessmentTargetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	input := &inspector.CreateAssessmentTargetInput{
		AssessmentTargetName: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("resource_group_arn"); ok {
		input.ResourceGroupArn = aws.String(v.(string))
	}

	resp, err := conn.CreateAssessmentTarget(input)
	if err != nil {
		return fmt.Errorf("error creating Inspector Assessment Target: %s", err)
	}

	d.SetId(*resp.AssessmentTargetArn)

	return resourceAwsInspectorAssessmentTargetRead(d, meta)
}

func resourceAwsInspectorAssessmentTargetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	assessmentTarget, err := describeInspectorAssessmentTarget(conn, d.Id())

	if err != nil {
		return fmt.Errorf("error describing Inspector Assessment Target (%s): %s", d.Id(), err)
	}

	if assessmentTarget == nil {
		log.Printf("[WARN] Inspector Assessment Target (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("arn", assessmentTarget.Arn)
	d.Set("name", assessmentTarget.Name)
	d.Set("resource_group_arn", assessmentTarget.ResourceGroupArn)

	return nil
}

func resourceAwsInspectorAssessmentTargetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	input := inspector.UpdateAssessmentTargetInput{
		AssessmentTargetArn:  aws.String(d.Id()),
		AssessmentTargetName: aws.String(d.Get("name").(string)),
	}

	if v, ok := d.GetOk("resource_group_arn"); ok {
		input.ResourceGroupArn = aws.String(v.(string))
	}

	_, err := conn.UpdateAssessmentTarget(&input)
	if err != nil {
		return fmt.Errorf("error updating Inspector Assessment Target (%s): %s", d.Id(), err)
	}

	return resourceAwsInspectorAssessmentTargetRead(d, meta)
}

func resourceAwsInspectorAssessmentTargetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	return resource.Retry(60*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteAssessmentTarget(&inspector.DeleteAssessmentTargetInput{
			AssessmentTargetArn: aws.String(d.Id()),
		})

		if isAWSErr(err, inspector.ErrCodeAssessmentRunInProgressException, "") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

}

func describeInspectorAssessmentTarget(conn *inspector.Inspector, arn string) (*inspector.AssessmentTarget, error) {
	input := &inspector.DescribeAssessmentTargetsInput{
		AssessmentTargetArns: []*string{aws.String(arn)},
	}

	output, err := conn.DescribeAssessmentTargets(input)

	if isAWSErr(err, inspector.ErrCodeInvalidInputException, "") {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	var assessmentTarget *inspector.AssessmentTarget
	for _, target := range output.AssessmentTargets {
		if aws.StringValue(target.Arn) == arn {
			assessmentTarget = target
			break
		}
	}

	return assessmentTarget, nil
}
