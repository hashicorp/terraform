package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAutoscalingAttachment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAutoscalingAttachmentCreate,
		Read:   resourceAwsAutoscalingAttachmentRead,
		Delete: resourceAwsAutoscalingAttachmentDelete,

		Schema: map[string]*schema.Schema{
			"autoscaling_group_name": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},

			"elb": &schema.Schema{
				Type:     schema.TypeString,
				ForceNew: true,
				Required: true,
			},
		},
	}
}

func resourceAwsAutoscalingAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	asgconn := meta.(*AWSClient).autoscalingconn
	asgName := d.Get("autoscaling_group_name").(string)
	elbName := d.Get("elb").(string)

	attachElbInput := &autoscaling.AttachLoadBalancersInput{
		AutoScalingGroupName: aws.String(asgName),
		LoadBalancerNames:    []*string{aws.String(elbName)},
	}

	log.Printf("[INFO] registering asg %s with ELBs %s", asgName, elbName)

	if _, err := asgconn.AttachLoadBalancers(attachElbInput); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Failure attaching AutoScaling Group %s with Elastic Load Balancer: %s: {{err}}", asgName, elbName), err)
	}

	d.SetId(resource.PrefixedUniqueId(fmt.Sprintf("%s-", asgName)))

	return resourceAwsAutoscalingAttachmentRead(d, meta)
}

func resourceAwsAutoscalingAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	asgconn := meta.(*AWSClient).autoscalingconn
	asgName := d.Get("autoscaling_group_name").(string)
	elbName := d.Get("elb").(string)

	// Retrieve the ASG properites to get list of associated ELBs
	asg, err := getAwsAutoscalingGroup(asgName, asgconn)

	if err != nil {
		return err
	}
	if asg == nil {
		log.Printf("[INFO] Autoscaling Group %q not found", asgName)
		d.SetId("")
		return nil
	}

	found := false
	for _, i := range asg.LoadBalancerNames {
		if elbName == *i {
			d.Set("elb", elbName)
			found = true
			break
		}
	}

	if !found {
		log.Printf("[WARN] Association for %s was not found in ASG assocation", elbName)
		d.SetId("")
	}

	return nil
}

func resourceAwsAutoscalingAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	asgconn := meta.(*AWSClient).autoscalingconn
	asgName := d.Get("autoscaling_group_name").(string)
	elbName := d.Get("elb").(string)

	log.Printf("[INFO] Deleting ELB %s association from: %s", elbName, asgName)

	detachOpts := &autoscaling.DetachLoadBalancersInput{
		AutoScalingGroupName: aws.String(asgName),
		LoadBalancerNames:    []*string{aws.String(elbName)},
	}

	if _, err := asgconn.DetachLoadBalancers(detachOpts); err != nil {
		return errwrap.Wrapf(fmt.Sprintf("Failure detaching AutoScaling Group %s with Elastic Load Balancer: %s: {{err}}", asgName, elbName), err)
	}

	return nil
}
