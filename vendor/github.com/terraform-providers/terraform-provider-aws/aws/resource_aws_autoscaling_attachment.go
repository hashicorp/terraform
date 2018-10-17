package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAutoscalingAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAutoscalingAttachmentCreate,
		Read:   resourceAwsAutoscalingAttachmentRead,
		Delete: resourceAwsAutoscalingAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"autoscaling_group_name": {
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"elb": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},

			"alb_target_group_arn": {
				Type:     schema.TypeString,
				ForceNew: true,
				Optional: true,
			},
		},
	}
}

func resourceAwsAutoscalingAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	asgconn := meta.(*AWSClient).autoscalingconn
	asgName := d.Get("autoscaling_group_name").(string)

	if v, ok := d.GetOk("elb"); ok {
		attachOpts := &autoscaling.AttachLoadBalancersInput{
			AutoScalingGroupName: aws.String(asgName),
			LoadBalancerNames:    []*string{aws.String(v.(string))},
		}

		log.Printf("[INFO] registering asg %s with ELBs %s", asgName, v.(string))

		if _, err := asgconn.AttachLoadBalancers(attachOpts); err != nil {
			return fmt.Errorf("Failure attaching AutoScaling Group %s with Elastic Load Balancer: %s: %s", asgName, v.(string), err)
		}
	}

	if v, ok := d.GetOk("alb_target_group_arn"); ok {
		attachOpts := &autoscaling.AttachLoadBalancerTargetGroupsInput{
			AutoScalingGroupName: aws.String(asgName),
			TargetGroupARNs:      []*string{aws.String(v.(string))},
		}

		log.Printf("[INFO] registering asg %s with ALB Target Group %s", asgName, v.(string))

		if _, err := asgconn.AttachLoadBalancerTargetGroups(attachOpts); err != nil {
			return fmt.Errorf("Failure attaching AutoScaling Group %s with ALB Target Group: %s: %s", asgName, v.(string), err)
		}
	}

	d.SetId(resource.PrefixedUniqueId(fmt.Sprintf("%s-", asgName)))

	return resourceAwsAutoscalingAttachmentRead(d, meta)
}

func resourceAwsAutoscalingAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	asgconn := meta.(*AWSClient).autoscalingconn
	asgName := d.Get("autoscaling_group_name").(string)

	// Retrieve the ASG properties to get list of associated ELBs
	asg, err := getAwsAutoscalingGroup(asgName, asgconn)

	if err != nil {
		return err
	}
	if asg == nil {
		log.Printf("[WARN] Autoscaling Group (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if v, ok := d.GetOk("elb"); ok {
		found := false
		for _, i := range asg.LoadBalancerNames {
			if v.(string) == *i {
				d.Set("elb", v.(string))
				found = true
				break
			}
		}

		if !found {
			log.Printf("[WARN] Association for %s was not found in ASG association", v.(string))
			d.SetId("")
		}
	}

	if v, ok := d.GetOk("alb_target_group_arn"); ok {
		found := false
		for _, i := range asg.TargetGroupARNs {
			if v.(string) == *i {
				d.Set("alb_target_group_arn", v.(string))
				found = true
				break
			}
		}

		if !found {
			log.Printf("[WARN] Association for %s was not found in ASG association", v.(string))
			d.SetId("")
		}
	}

	return nil
}

func resourceAwsAutoscalingAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	asgconn := meta.(*AWSClient).autoscalingconn
	asgName := d.Get("autoscaling_group_name").(string)

	if v, ok := d.GetOk("elb"); ok {
		detachOpts := &autoscaling.DetachLoadBalancersInput{
			AutoScalingGroupName: aws.String(asgName),
			LoadBalancerNames:    []*string{aws.String(v.(string))},
		}

		log.Printf("[INFO] Deleting ELB %s association from: %s", v.(string), asgName)
		if _, err := asgconn.DetachLoadBalancers(detachOpts); err != nil {
			return fmt.Errorf("Failure detaching AutoScaling Group %s with Elastic Load Balancer: %s: %s", asgName, v.(string), err)
		}
	}

	if v, ok := d.GetOk("alb_target_group_arn"); ok {
		detachOpts := &autoscaling.DetachLoadBalancerTargetGroupsInput{
			AutoScalingGroupName: aws.String(asgName),
			TargetGroupARNs:      []*string{aws.String(v.(string))},
		}

		log.Printf("[INFO] Deleting ALB Target Group %s association from: %s", v.(string), asgName)
		if _, err := asgconn.DetachLoadBalancerTargetGroups(detachOpts); err != nil {
			return fmt.Errorf("Failure detaching AutoScaling Group %s with ALB Target Group: %s: %s", asgName, v.(string), err)
		}
	}

	return nil
}
