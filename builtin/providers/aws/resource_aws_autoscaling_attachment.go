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
		Update: resourceAwsAutoscalingAttachmentUpdate,
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

			"instance_ids": {
				Type:          schema.TypeList,
				Optional:      true,
				ConflictsWith: []string{"elb"},
				Elem:          &schema.Schema{Type: schema.TypeString},
			},

			"should_change_capacity": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
		},
	}
}

func diffAsgAttachmentInstanceIds(oldInstanceIds, newInstanceIds []interface{}) ([]interface{}, []interface{}) {
	create := make([]interface{}, len(newInstanceIds))
	for i, t := range newInstanceIds {
		create[i] = t
	}

	remove := make([]interface{}, 0)
	for i, t := range oldInstanceIds {
		if create[i] != t {
			remove = append(remove, t)
		}
	}

	return create, remove
}

func resourceAwsAutoscalingAttachmentUpdate(d *schema.ResourceData, meta interface{}) error {
	asgconn := meta.(*AWSClient).autoscalingconn

	asgName := d.Get("autoscaling_group_name").(string)

	if d.HasChange("instance_ids") {
		oraw, nraw := d.GetChange("instance_ids")
		o := oraw.([]interface{})
		n := nraw.([]interface{})
		create, remove := diffAsgAttachmentInstanceIds(o, n)

		if len(remove) > 0 {

			detachInstancesInput := &autoscaling.DetachInstancesInput{
				AutoScalingGroupName:           aws.String(asgName),
				InstanceIds:                    expandStringList(create),
				ShouldDecrementDesiredCapacity: aws.Bool(d.Get("should_change_capacity").(bool)),
			}

			log.Printf("[INFO] detaching asg %s from Instances", asgName)

			if _, err := asgconn.DetachInstances(detachInstancesInput); err != nil {
				return errwrap.Wrapf(fmt.Sprintf("Failure detaching AutoScaling Group %s from Instances: {{err}}", asgName), err)
			}
		}

		if len(create) > 0 {
			attachInstancesInput := &autoscaling.AttachInstancesInput{
				AutoScalingGroupName: aws.String(asgName),
				InstanceIds:          expandStringList(create),
			}

			log.Printf("[INFO] registering asg %s with Instances", asgName)

			if _, err := asgconn.AttachInstances(attachInstancesInput); err != nil {
				return errwrap.Wrapf(fmt.Sprintf("Failure attaching AutoScaling Group %s with Instances: {{err}}", asgName), err)
			}
		}
	}

	return nil
}

func resourceAwsAutoscalingAttachmentCreate(d *schema.ResourceData, meta interface{}) error {
	asgconn := meta.(*AWSClient).autoscalingconn
	asgName := d.Get("autoscaling_group_name").(string)

	if v, ok := d.GetOk("elb"); ok {

		attachElbInput := &autoscaling.AttachLoadBalancersInput{
			AutoScalingGroupName: aws.String(asgName),
			LoadBalancerNames:    []*string{aws.String(v.(string))},
		}

		log.Printf("[INFO] registering asg %s with ELBs %s", asgName, v.(string))

		if _, err := asgconn.AttachLoadBalancers(attachElbInput); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Failure attaching AutoScaling Group %s with Elastic Load Balancer: %s: {{err}}", asgName, v.(string)), err)
		}
	}

	if v, ok := d.GetOk("instance_ids"); ok {

		attachInstancesInput := &autoscaling.AttachInstancesInput{
			AutoScalingGroupName: aws.String(asgName),
			InstanceIds:          expandStringList(v.([]interface{})),
		}

		log.Printf("[INFO] registering asg %s with Instances", asgName)

		if _, err := asgconn.AttachInstances(attachInstancesInput); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Failure attaching AutoScaling Group %s with Instances: {{err}}", asgName), err)
		}
	}

	d.SetId(resource.PrefixedUniqueId(fmt.Sprintf("%s-", asgName)))

	return resourceAwsAutoscalingAttachmentRead(d, meta)
}

func resourceAwsAutoscalingAttachmentRead(d *schema.ResourceData, meta interface{}) error {
	asgconn := meta.(*AWSClient).autoscalingconn
	asgName := d.Get("autoscaling_group_name").(string)

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
			log.Printf("[WARN] Association for %s was not found in ASG assocation", v.(string))
			d.SetId("")
		}
	}

	if v, ok := d.GetOk("instance_ids"); ok {
		target := len(v.([]interface{}))
		var actual int
		var instanceIds []string

		for _, i := range asg.Instances {
			for _, e := range v.([]interface{}) {
				if e.(string) == *i.InstanceId {
					actual += 1
					instanceIds = append(instanceIds, *i.InstanceId)
				}
			}
		}

		if target != actual {
			log.Printf("[WARN] Expected %d instances in the ASG association, got %d", target, actual)
			d.SetId("")
		} else {
			d.Set("instance_ids", instanceIds)
		}
	}

	return nil
}

func resourceAwsAutoscalingAttachmentDelete(d *schema.ResourceData, meta interface{}) error {
	asgconn := meta.(*AWSClient).autoscalingconn
	asgName := d.Get("autoscaling_group_name").(string)

	if v, ok := d.GetOk("elb"); ok {
		log.Printf("[INFO] Deleting ELB %s association from: %s", v.(string), asgName)

		detachOpts := &autoscaling.DetachLoadBalancersInput{
			AutoScalingGroupName: aws.String(asgName),
			LoadBalancerNames:    []*string{aws.String(v.(string))},
		}

		if _, err := asgconn.DetachLoadBalancers(detachOpts); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Failure detaching AutoScaling Group %s with Elastic Load Balancer: %s: {{err}}", asgName, v.(string)), err)
		}
	}

	if v, ok := d.GetOk("instance_ids"); ok {
		log.Printf("[INFO] Deleting InstanceIds from ASG association from: %s", asgName)

		detachOpts := &autoscaling.DetachInstancesInput{
			AutoScalingGroupName:           aws.String(asgName),
			InstanceIds:                    expandStringList(v.([]interface{})),
			ShouldDecrementDesiredCapacity: aws.Bool(d.Get("should_change_capacity").(bool)),
		}

		if _, err := asgconn.DetachInstances(detachOpts); err != nil {
			return errwrap.Wrapf(fmt.Sprintf("Failure detaching AutoScaling Group %s from Instances: {{err}}", asgName), err)
		}
	}

	return nil
}
