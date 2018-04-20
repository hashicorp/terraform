package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ssm"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSsmMaintenanceWindowTask() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSsmMaintenanceWindowTaskCreate,
		Read:   resourceAwsSsmMaintenanceWindowTaskRead,
		Delete: resourceAwsSsmMaintenanceWindowTaskDelete,

		Schema: map[string]*schema.Schema{
			"window_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"max_concurrency": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"max_errors": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"task_type": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"task_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"service_role_arn": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"targets": {
				Type:     schema.TypeList,
				Required: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"values": {
							Type:     schema.TypeList,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"priority": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"logging_info": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"s3_bucket_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"s3_region": {
							Type:     schema.TypeString,
							Required: true,
						},
						"s3_bucket_prefix": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"task_parameters": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
						"values": {
							Type:     schema.TypeList,
							Required: true,
							ForceNew: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
		},
	}
}

func expandAwsSsmMaintenanceWindowLoggingInfo(config []interface{}) *ssm.LoggingInfo {

	loggingConfig := config[0].(map[string]interface{})

	loggingInfo := &ssm.LoggingInfo{
		S3BucketName: aws.String(loggingConfig["s3_bucket_name"].(string)),
		S3Region:     aws.String(loggingConfig["s3_region"].(string)),
	}

	if s := loggingConfig["s3_bucket_prefix"].(string); s != "" {
		loggingInfo.S3KeyPrefix = aws.String(s)
	}

	return loggingInfo
}

func flattenAwsSsmMaintenanceWindowLoggingInfo(loggingInfo *ssm.LoggingInfo) []interface{} {

	result := make(map[string]interface{})
	result["s3_bucket_name"] = *loggingInfo.S3BucketName
	result["s3_region"] = *loggingInfo.S3Region

	if loggingInfo.S3KeyPrefix != nil {
		result["s3_bucket_prefix"] = *loggingInfo.S3KeyPrefix
	}

	return []interface{}{result}
}

func expandAwsSsmTaskParameters(config []interface{}) map[string]*ssm.MaintenanceWindowTaskParameterValueExpression {
	params := make(map[string]*ssm.MaintenanceWindowTaskParameterValueExpression)
	for _, v := range config {
		paramConfig := v.(map[string]interface{})
		params[paramConfig["name"].(string)] = &ssm.MaintenanceWindowTaskParameterValueExpression{
			Values: expandStringList(paramConfig["values"].([]interface{})),
		}
	}
	return params
}

func flattenAwsSsmTaskParameters(taskParameters map[string]*ssm.MaintenanceWindowTaskParameterValueExpression) []interface{} {
	result := make([]interface{}, 0, len(taskParameters))
	for k, v := range taskParameters {
		taskParam := map[string]interface{}{
			"name":   k,
			"values": flattenStringList(v.Values),
		}
		result = append(result, taskParam)
	}

	return result
}

func resourceAwsSsmMaintenanceWindowTaskCreate(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Registering SSM Maintenance Window Task")

	params := &ssm.RegisterTaskWithMaintenanceWindowInput{
		WindowId:       aws.String(d.Get("window_id").(string)),
		MaxConcurrency: aws.String(d.Get("max_concurrency").(string)),
		MaxErrors:      aws.String(d.Get("max_errors").(string)),
		TaskType:       aws.String(d.Get("task_type").(string)),
		ServiceRoleArn: aws.String(d.Get("service_role_arn").(string)),
		TaskArn:        aws.String(d.Get("task_arn").(string)),
		Targets:        expandAwsSsmTargets(d.Get("targets").([]interface{})),
	}

	if v, ok := d.GetOk("priority"); ok {
		params.Priority = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("logging_info"); ok {
		params.LoggingInfo = expandAwsSsmMaintenanceWindowLoggingInfo(v.([]interface{}))
	}

	if v, ok := d.GetOk("task_parameters"); ok {
		params.TaskParameters = expandAwsSsmTaskParameters(v.([]interface{}))
	}

	resp, err := ssmconn.RegisterTaskWithMaintenanceWindow(params)
	if err != nil {
		return err
	}

	d.SetId(*resp.WindowTaskId)

	return resourceAwsSsmMaintenanceWindowTaskRead(d, meta)
}

func resourceAwsSsmMaintenanceWindowTaskRead(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	params := &ssm.DescribeMaintenanceWindowTasksInput{
		WindowId: aws.String(d.Get("window_id").(string)),
	}

	resp, err := ssmconn.DescribeMaintenanceWindowTasks(params)
	if err != nil {
		return err
	}

	found := false
	for _, t := range resp.Tasks {
		if *t.WindowTaskId == d.Id() {
			found = true

			d.Set("window_id", t.WindowId)
			d.Set("max_concurrency", t.MaxConcurrency)
			d.Set("max_errors", t.MaxErrors)
			d.Set("task_type", t.Type)
			d.Set("service_role_arn", t.ServiceRoleArn)
			d.Set("task_arn", t.TaskArn)
			d.Set("priority", t.Priority)

			if t.LoggingInfo != nil {
				if err := d.Set("logging_info", flattenAwsSsmMaintenanceWindowLoggingInfo(t.LoggingInfo)); err != nil {
					return fmt.Errorf("[DEBUG] Error setting logging_info error: %#v", err)
				}
			}

			if t.TaskParameters != nil {
				if err := d.Set("task_parameters", flattenAwsSsmTaskParameters(t.TaskParameters)); err != nil {
					return fmt.Errorf("[DEBUG] Error setting task_parameters error: %#v", err)
				}
			}

			if err := d.Set("targets", flattenAwsSsmTargets(t.Targets)); err != nil {
				return fmt.Errorf("[DEBUG] Error setting targets error: %#v", err)
			}
		}
	}

	if !found {
		log.Printf("[INFO] Maintenance Window Target not found. Removing from state")
		d.SetId("")
		return nil
	}

	return nil
}

func resourceAwsSsmMaintenanceWindowTaskDelete(d *schema.ResourceData, meta interface{}) error {
	ssmconn := meta.(*AWSClient).ssmconn

	log.Printf("[INFO] Deregistering SSM Maintenance Window Task: %s", d.Id())

	params := &ssm.DeregisterTaskFromMaintenanceWindowInput{
		WindowId:     aws.String(d.Get("window_id").(string)),
		WindowTaskId: aws.String(d.Id()),
	}

	_, err := ssmconn.DeregisterTaskFromMaintenanceWindow(params)
	if err != nil {
		return err
	}

	return nil
}
