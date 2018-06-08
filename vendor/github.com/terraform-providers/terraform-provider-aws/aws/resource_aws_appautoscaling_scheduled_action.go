package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

const awsAppautoscalingScheduleTimeLayout = "2006-01-02T15:04:05Z"

func resourceAwsAppautoscalingScheduledAction() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsAppautoscalingScheduledActionPut,
		Read:   resourceAwsAppautoscalingScheduledActionRead,
		Delete: resourceAwsAppautoscalingScheduledActionDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_namespace": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"resource_id": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"scalable_dimension": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"scalable_target_action": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_capacity": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"min_capacity": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
					},
				},
			},
			"schedule": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"start_time": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"end_time": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsAppautoscalingScheduledActionPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appautoscalingconn

	input := &applicationautoscaling.PutScheduledActionInput{
		ScheduledActionName: aws.String(d.Get("name").(string)),
		ServiceNamespace:    aws.String(d.Get("service_namespace").(string)),
		ResourceId:          aws.String(d.Get("resource_id").(string)),
	}
	if v, ok := d.GetOk("scalable_dimension"); ok {
		input.ScalableDimension = aws.String(v.(string))
	}
	if v, ok := d.GetOk("schedule"); ok {
		input.Schedule = aws.String(v.(string))
	}
	if v, ok := d.GetOk("scalable_target_action"); ok {
		sta := &applicationautoscaling.ScalableTargetAction{}
		raw := v.([]interface{})[0].(map[string]interface{})
		if max, ok := raw["max_capacity"]; ok {
			sta.MaxCapacity = aws.Int64(int64(max.(int)))
		}
		if min, ok := raw["min_capacity"]; ok {
			sta.MinCapacity = aws.Int64(int64(min.(int)))
		}
		input.ScalableTargetAction = sta
	}
	if v, ok := d.GetOk("start_time"); ok {
		t, err := time.Parse(awsAppautoscalingScheduleTimeLayout, v.(string))
		if err != nil {
			return fmt.Errorf("Error Parsing Appautoscaling Scheduled Action Start Time: %s", err.Error())
		}
		input.StartTime = aws.Time(t)
	}
	if v, ok := d.GetOk("end_time"); ok {
		t, err := time.Parse(awsAppautoscalingScheduleTimeLayout, v.(string))
		if err != nil {
			return fmt.Errorf("Error Parsing Appautoscaling Scheduled Action End Time: %s", err.Error())
		}
		input.EndTime = aws.Time(t)
	}

	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.PutScheduledAction(input)
		if err != nil {
			if isAWSErr(err, applicationautoscaling.ErrCodeObjectNotFoundException, "") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return err
	}

	d.SetId(d.Get("name").(string) + "-" + d.Get("service_namespace").(string) + "-" + d.Get("resource_id").(string))
	return resourceAwsAppautoscalingScheduledActionRead(d, meta)
}

func resourceAwsAppautoscalingScheduledActionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appautoscalingconn

	saName := d.Get("name").(string)
	input := &applicationautoscaling.DescribeScheduledActionsInput{
		ScheduledActionNames: []*string{aws.String(saName)},
		ServiceNamespace:     aws.String(d.Get("service_namespace").(string)),
	}
	resp, err := conn.DescribeScheduledActions(input)
	if err != nil {
		return err
	}
	if len(resp.ScheduledActions) < 1 {
		log.Printf("[WARN] Application Autoscaling Scheduled Action (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}
	if len(resp.ScheduledActions) != 1 {
		return fmt.Errorf("Expected 1 scheduled action under %s, found %d", saName, len(resp.ScheduledActions))
	}
	if *resp.ScheduledActions[0].ScheduledActionName != saName {
		return fmt.Errorf("Scheduled Action (%s) not found", saName)
	}
	d.Set("arn", resp.ScheduledActions[0].ScheduledActionARN)
	return nil
}

func resourceAwsAppautoscalingScheduledActionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).appautoscalingconn

	input := &applicationautoscaling.DeleteScheduledActionInput{
		ScheduledActionName: aws.String(d.Get("name").(string)),
		ServiceNamespace:    aws.String(d.Get("service_namespace").(string)),
		ResourceId:          aws.String(d.Get("resource_id").(string)),
	}
	if v, ok := d.GetOk("scalable_dimension"); ok {
		input.ScalableDimension = aws.String(v.(string))
	}
	_, err := conn.DeleteScheduledAction(input)
	if err != nil {
		if isAWSErr(err, applicationautoscaling.ErrCodeObjectNotFoundException, "") {
			log.Printf("[WARN] Application Autoscaling Scheduled Action (%s) already gone, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return err
	}
	d.SetId("")
	return nil
}
