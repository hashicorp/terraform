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

func resourceAwsAlbTargetGroupAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAlbAttachmentCreate,
		Read:   resourceAwsAlbAttachmentRead,
		Delete: resourceAwsAlbAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"target_group_arn": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"target_id": {
				Type:          schema.TypeString,
				ForceNew:      true,
				Optional:      true,
				Deprecated:    "Use field target instead",
				ConflictsWith: []string{"target"},
			},

			"port": {
				Type:          schema.TypeInt,
				ForceNew:      true,
				Optional:      true,
				Deprecated:    "Use field target instead",
				ConflictsWith: []string{"target"},
			},

			"target": {
				Type:          schema.TypeSet,
				Optional:      true,
				ForceNew:      true,
				ConflictsWith: []string{"target_id", "port"},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"target_id": {
							Type:     schema.TypeString,
							ForceNew: true,
							Required: true,
						},

						"port": {
							Type:     schema.TypeInt,
							ForceNew: true,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsAlbAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	params := &elbv2.RegisterTargetsInput{
		TargetGroupArn: aws.String(d.Get("target_group_arn").(string)),
	}

	if _, ok := d.GetOk("target_id"); ok {
		targetPortToAdd := &elbv2.TargetDescription{
			Id:   aws.String(d.Get("target_id").(string)),
			Port: aws.Int64(int64(d.Get("port").(int))),
		}
		params.Targets = append(params.Targets, targetPortToAdd)
		log.Printf("[INFO] Registering Target %s (%d) with Target Group %s", d.Get("target_id").(string),
			d.Get("port").(int), d.Get("target_group_arn").(string))
	} else {

		targets := d.Get("target").(*schema.Set)
		for _, target := range targets.List() {
			tp := target.(map[string]interface{})
			targetPortToAdd := &elbv2.TargetDescription{
				Id:   aws.String(tp["target_id"].(string)),
				Port: aws.Int64(int64(tp["port"].(int))),
			}
			params.Targets = append(params.Targets, targetPortToAdd)

		}

	}

	_, err := elbconn.RegisterTargets(params)
	if err != nil {
		return errwrap.Wrapf("Error registering targets with target group: {{err}}", err)
	}

	d.SetId(resource.PrefixedUniqueId(fmt.Sprintf("%s-", d.Get("target_group_arn"))))

	return nil
}

func resourceAwsAlbAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	params := &elbv2.DeregisterTargetsInput{
		TargetGroupArn: aws.String(d.Get("target_group_arn").(string)),
	}

	if _, ok := d.GetOk("target_id"); ok {
		targetPortToAdd := &elbv2.TargetDescription{
			Id:   aws.String(d.Get("target_id").(string)),
			Port: aws.Int64(int64(d.Get("port").(int))),
		}
		params.Targets = append(params.Targets, targetPortToAdd)
		log.Printf("[INFO] Registering Target %s (%d) with Target Group %s", d.Get("target_id").(string),
			d.Get("port").(int), d.Get("target_group_arn").(string))
	} else {

		targets := d.Get("target").(*schema.Set)
		for _, target := range targets.List() {
			tp := target.(map[string]interface{})
			targetPortToAdd := &elbv2.TargetDescription{
				Id:   aws.String(tp["target_id"].(string)),
				Port: aws.Int64(int64(tp["port"].(int))),
			}
			params.Targets = append(params.Targets, targetPortToAdd)

		}

	}

	_, err := elbconn.DeregisterTargets(params)
	if err != nil && !isTargetGroupNotFound(err) {
		return errwrap.Wrapf("Error deregistering Targets: {{err}}", err)
	}

	d.SetId("")

	return nil
}

// resourceAwsAlbAttachmentRead requires all of the fields in order to describe the correct
// target, so there is no work to do beyond ensuring that the target and group still exist.
func resourceAwsAlbAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	elbconn := meta.(*AWSClient).elbv2conn

	params := &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(d.Get("target_group_arn").(string)),
	}

	if _, ok := d.GetOk("target_id"); ok {
		targetPortToAdd := &elbv2.TargetDescription{
			Id:   aws.String(d.Get("target_id").(string)),
			Port: aws.Int64(int64(d.Get("port").(int))),
		}
		params.Targets = append(params.Targets, targetPortToAdd)
		log.Printf("[INFO] Registering Target %s (%d) with Target Group %s", d.Get("target_id").(string),
			d.Get("port").(int), d.Get("target_group_arn").(string))
	} else {

		targets := d.Get("target").(*schema.Set)
		for _, target := range targets.List() {
			tp := target.(map[string]interface{})
			targetPortToAdd := &elbv2.TargetDescription{
				Id:   aws.String(tp["target_id"].(string)),
				Port: aws.Int64(int64(tp["port"].(int))),
			}
			params.Targets = append(params.Targets, targetPortToAdd)

		}

	}

	resp, err := elbconn.DescribeTargetHealth(params)

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
