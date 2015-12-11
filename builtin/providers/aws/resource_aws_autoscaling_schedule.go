package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsAutoscalingSchedule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAutoscalingScheduleCreate,
		Read:   resourceAwsAutoscalingScheduleRead,
		Update: resourceAwsAutoscalingScheduleCreate,
		Delete: resourceAwsAutoscalingScheduleDelete,

		Schema: map[string]*schema.Schema{
			"scheduled_action_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"autoscaling_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"start_time": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"end_time": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"recurrence": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"min_size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"max_size": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"desired_capacity": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsAutoscalingScheduleCreate(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn
	params := autoscaling.PutScheduledUpdateGroupActionInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		ScheduledActionName:  aws.String(d.Get("scheduled_action_name").(string)),
	}

	if attr, ok := d.GetOk("start_time"); ok {
		params.StartTime = aws.Time()
	}

	if attr, ok := d.GetOk("min_size"); ok {
		params.MinSize = aws.Int(int64(attr.(int)))
	}

	if attr, ok := d.GetOk("max_size"); ok {
		params.MaxSize = aws.Int(int64(attr.(int)))
	}

	if attr, ok := d.GetOk("desired_capacity"); ok {
		params.DesiredCapacity = aws.Int(int64(attr.(int)))
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
	return nil
}

func resourceAwsAutoscalingScheduleDelete(d *schema.ResourceData, meta interface{}) error {
	autoscalingconn := meta.(*AWSClient).autoscalingconn

	params := autoscaling.DeleteScheduledActionInput{
		AutoScalingGroupName: aws.String(d.Get("autoscaling_group_name").(string)),
		ScheduledActionName:  aws.String(d.Get("scheduled_action_name").(string)),
	}

	log.Printf("[INFO] Deleting Autoscaling Scheduled Action: %s", d.Get("scheduled_action_name").(string))
	_, err := autoscalingconn.DeleteScheduledAction(params)
	if err != nil {
		return fmt.Errorf("Error deleting Autoscaling Scheduled Action: %s", err.Error())
	}

	return nil
}
