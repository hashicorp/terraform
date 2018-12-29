package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/schema"
)

const awsAutoscalingScheduleTimeLayout = "2006-01-02T15:04:05Z"

func resourceAwsAutoscalingSchedule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAutoscalingScheduleCreate,
		Read:   resourceAwsAutoscalingScheduleRead,
		Update: resourceAwsAutoscalingScheduleCreate,
		Delete: resourceAwsAutoscalingScheduleDelete,

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"scheduled_action_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"autoscaling_group_name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"start_time": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateASGScheduleTimestamp,
			},
			"end_time": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateASGScheduleTimestamp,
			},
			"recurrence": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"min_size": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"max_size": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"desired_capacity": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsAutoscalingScheduleCreate(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn
	params := &autoscaling.PutScheduledUpdateGroupActionInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		ScheduledActionName:  aws.String(d.Get("scheduled_action_name").(string)),
	}

	if attr, ok := d.GetOk("start_time"); ok {
		t, err := time.Parse(awsAutoscalingScheduleTimeLayout, attr.(string))
		if err != nil {
			return fmt.Errorf("Error Parsing AWS Autoscaling Group Schedule Start Time: %s", err.Error())
		}
		params.StartTime = aws.Time(t)
	}

	if attr, ok := d.GetOk("end_time"); ok {
		t, err := time.Parse(awsAutoscalingScheduleTimeLayout, attr.(string))
		if err != nil {
			return fmt.Errorf("Error Parsing AWS Autoscaling Group Schedule End Time: %s", err.Error())
		}
		params.EndTime = aws.Time(t)
	}

	if attr, ok := d.GetOk("recurrence"); ok {
		params.Recurrence = aws.String(attr.(string))
	}

	// Scheduled actions don't need to set all three size parameters. For example,
	// you may want to change the min or max without also forcing an immediate
	// resize by changing a desired_capacity that may have changed due to other
	// autoscaling rules. Since Terraform doesn't have a great pattern for
	// differentiating between 0 and unset fields, we accept "-1" to mean "don't
	// include this parameter in the action".
	minSize := int64(d.Get("min_size").(int))
	maxSize := int64(d.Get("max_size").(int))
	desiredCapacity := int64(d.Get("desired_capacity").(int))
	if minSize != -1 {
		params.MinSize = aws.Int64(minSize)
	}
	if maxSize != -1 {
		params.MaxSize = aws.Int64(maxSize)
	}
	if desiredCapacity != -1 {
		params.DesiredCapacity = aws.Int64(desiredCapacity)
	}

	log.Printf("[INFO] Creating Autoscaling Scheduled Action: %s", d.Get("scheduled_action_name").(string))
	_, err := autoscalingconn.PutScheduledUpdateGroupAction(params)
	if err != nil {
		return fmt.Errorf("Error Creating Autoscaling Scheduled Action: %s", err.Error())
	}

	d.SetId(d.Get("scheduled_action_name").(string))

	return resourceAwsAutoscalingScheduleRead(d, meta)
}

func resourceAwsAutoscalingScheduleRead(d *schema.ResourceData, meta interface{}) error {
	sa, err, exists := resourceAwsASGScheduledActionRetrieve(d, meta)
	if err != nil {
		return err
	}

	if !exists {
		log.Printf("[WARN] Autoscaling Scheduled Action (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("autoscaling_group_name", sa.AutoScalingGroupName)
	d.Set("arn", sa.ScheduledActionARN)

	if sa.MinSize == nil {
		d.Set("min_size", -1)
	} else {
		d.Set("min_size", sa.MinSize)
	}
	if sa.MaxSize == nil {
		d.Set("max_size", -1)
	} else {
		d.Set("max_size", sa.MaxSize)
	}
	if sa.DesiredCapacity == nil {
		d.Set("desired_capacity", -1)
	} else {
		d.Set("desired_capacity", sa.DesiredCapacity)
	}

	d.Set("recurrence", sa.Recurrence)

	if sa.StartTime != nil {
		d.Set("start_time", sa.StartTime.Format(awsAutoscalingScheduleTimeLayout))
	}

	if sa.EndTime != nil {
		d.Set("end_time", sa.EndTime.Format(awsAutoscalingScheduleTimeLayout))
	}

	return nil
}

func resourceAwsAutoscalingScheduleDelete(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	params := &autoscaling.DeleteScheduledActionInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		ScheduledActionName:  aws.String(d.Id()),
	}

	log.Printf("[INFO] Deleting Autoscaling Scheduled Action: %s", d.Id())
	_, err := autoscalingconn.DeleteScheduledAction(params)
	if err != nil {
		return fmt.Errorf("Error deleting Autoscaling Scheduled Action: %s", err.Error())
	}

	return nil
}

func resourceAwsASGScheduledActionRetrieve(d *schema.ResourceData, meta interface{}) (*autoscaling.ScheduledUpdateGroupAction, error, bool) {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	params := &autoscaling.DescribeScheduledActionsInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		ScheduledActionNames: []*string{aws.String(d.Id())},
	}

	log.Printf("[INFO] Describing Autoscaling Scheduled Action: %+v", params)
	actions, err := autoscalingconn.DescribeScheduledActions(params)
	if err != nil {
		//A ValidationError here can mean that either the Schedule is missing OR the Autoscaling Group is missing
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "ValidationError" {
			log.Printf("[WARN] Autoscaling Scheduled Action (%s) not found, removing from state", d.Id())
			d.SetId("")

			return nil, nil, false
		}
		return nil, fmt.Errorf("Error retrieving Autoscaling Scheduled Actions: %s", err), false
	}

	if len(actions.ScheduledUpdateGroupActions) != 1 ||
		*actions.ScheduledUpdateGroupActions[0].ScheduledActionName != d.Id() {
		return nil, nil, false
	}

	return actions.ScheduledUpdateGroupActions[0], nil, true
}
