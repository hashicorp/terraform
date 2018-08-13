package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsLbTargetGroupAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsLbAttachmentCreate,
		Read:   resourceAwsLbAttachmentRead,
		Delete: resourceAwsLbAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"target_group_arn": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"target_id": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"port": {
				Type:     schema.TypeInt,
				ForceNew: true,
				Optional: true,
			},

			"availability_zone": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
		},
	}
}

func resourceAwsLbAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	target := &elbv2.TargetDescription{
		Id: aws.String(d.Get("target_id").(string)),
	}

	if v, ok := d.GetOk("port"); ok {
		target.Port = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("availability_zone"); ok {
		target.AvailabilityZone = aws.String(v.(string))
	}

	params := &elbv2.RegisterTargetsInput{
		TargetGroupArn: aws.String(d.Get("target_group_arn").(string)),
		Targets:        []*elbv2.TargetDescription{target},
	}

	log.Printf("[INFO] Registering Target %s with Target Group %s", d.Get("target_id").(string),
		d.Get("target_group_arn").(string))

	_, err := elbconn.RegisterTargets(params)
	if err != nil {
		return errwrap.Wrapf("Error registering targets with target group: {{err}}", err)
	}

	d.SetId(resource.PrefixedUniqueId(fmt.Sprintf("%s-", d.Get("target_group_arn"))))

	return nil
}

func resourceAwsLbAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	target := &elbv2.TargetDescription{
		Id: aws.String(d.Get("target_id").(string)),
	}

	if v, ok := d.GetOk("port"); ok {
		target.Port = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("availability_zone"); ok {
		target.AvailabilityZone = aws.String(v.(string))
	}

	params := &elbv2.DeregisterTargetsInput{
		TargetGroupArn: aws.String(d.Get("target_group_arn").(string)),
		Targets:        []*elbv2.TargetDescription{target},
	}

	_, err := elbconn.DeregisterTargets(params)
	if err != nil && !isTargetGroupNotFound(err) {
		return errwrap.Wrapf("Error deregistering Targets: {{err}}", err)
	}

	return nil
}

// resourceAwsLbAttachmentRead requires all of the fields in order to describe the correct
// target, so there is no work to do beyond ensuring that the target and group still exist.
func resourceAwsLbAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	target := &elbv2.TargetDescription{
		Id: aws.String(d.Get("target_id").(string)),
	}

	if v, ok := d.GetOk("port"); ok {
		target.Port = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("availability_zone"); ok {
		target.AvailabilityZone = aws.String(v.(string))
	}

	resp, err := elbconn.DescribeTargetHealth(&elbv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(d.Get("target_group_arn").(string)),
		Targets:        []*elbv2.TargetDescription{target},
	})
	if err != nil {
		if isTargetGroupNotFound(err) {
			log.Printf("[WARN] Target group does not exist, removing target attachment %s", d.Id())
			d.SetId("")
			return nil
		}
		if isInvalidTarget(err) {
			log.Printf("[WARN] Target does not exist, removing target attachment %s", d.Id())
			d.SetId("")
			return nil
		}
		return errwrap.Wrapf("Error reading Target Health: {{err}}", err)
	}

	if len(resp.TargetHealthDescriptions) != 1 {
		log.Printf("[WARN] Target does not exist, removing target attachment %s", d.Id())
		d.SetId("")
		return nil
	}

	return nil
}

func isInvalidTarget(err error) bool {
	elberr, ok := err.(awserr.Error)
	return ok && elberr.Code() == "InvalidTarget"
}
