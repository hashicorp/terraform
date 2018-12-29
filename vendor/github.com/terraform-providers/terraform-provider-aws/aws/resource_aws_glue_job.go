package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/glue"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsGlueJob() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsGlueJobCreate,
		Read:   resourceAwsGlueJobRead,
		Update: resourceAwsGlueJobUpdate,
		Delete: resourceAwsGlueJobDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"allocated_capacity": {
				Type:         schema.TypeInt,
				Optional:     true,
				Default:      10,
				ValidateFunc: validation.IntAtLeast(2),
			},
			"command": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Optional: true,
							Default:  "glueetl",
						},
						"script_location": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
			"connections": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"default_arguments": {
				Type:     schema.TypeMap,
				Optional: true,
			},
			"description": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"execution_property": {
				Type:     schema.TypeList,
				Optional: true,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"max_concurrent_runs": {
							Type:         schema.TypeInt,
							Optional:     true,
							Default:      1,
							ValidateFunc: validation.IntAtLeast(1),
						},
					},
				},
			},
			"max_retries": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntBetween(0, 10),
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.NoZeroValues,
			},
			"role_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArn,
			},
			"timeout": {
				Type:     schema.TypeInt,
				Optional: true,
				Default:  2880,
			},
		},
	}
}

func resourceAwsGlueJobCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glueconn
	name := d.Get("name").(string)

	input := &glue.CreateJobInput{
		Command: expandGlueJobCommand(d.Get("command").([]interface{})),
		Name:    aws.String(name),
		Role:    aws.String(d.Get("role_arn").(string)),
		Timeout: aws.Int64(int64(d.Get("timeout").(int))),
	}

	if v, ok := d.GetOk("allocated_capacity"); ok {
		input.AllocatedCapacity = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("connections"); ok {
		input.Connections = &glue.ConnectionsList{
			Connections: expandStringList(v.([]interface{})),
		}
	}

	if kv, ok := d.GetOk("default_arguments"); ok {
		defaultArgumentsMap := make(map[string]string)
		for k, v := range kv.(map[string]interface{}) {
			defaultArgumentsMap[k] = v.(string)
		}
		input.DefaultArguments = aws.StringMap(defaultArgumentsMap)
	}

	if v, ok := d.GetOk("description"); ok {
		input.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("execution_property"); ok {
		input.ExecutionProperty = expandGlueExecutionProperty(v.([]interface{}))
	}

	if v, ok := d.GetOk("max_retries"); ok {
		input.MaxRetries = aws.Int64(int64(v.(int)))
	}

	log.Printf("[DEBUG] Creating Glue Job: %s", input)
	_, err := conn.CreateJob(input)
	if err != nil {
		return fmt.Errorf("error creating Glue Job (%s): %s", name, err)
	}

	d.SetId(name)

	return resourceAwsGlueJobRead(d, meta)
}

func resourceAwsGlueJobRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glueconn

	input := &glue.GetJobInput{
		JobName: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Reading Glue Job: %s", input)
	output, err := conn.GetJob(input)
	if err != nil {
		if isAWSErr(err, glue.ErrCodeEntityNotFoundException, "") {
			log.Printf("[WARN] Glue Job (%s) not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("error reading Glue Job (%s): %s", d.Id(), err)
	}

	job := output.Job
	if job == nil {
		log.Printf("[WARN] Glue Job (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("allocated_capacity", int(aws.Int64Value(job.AllocatedCapacity)))
	if err := d.Set("command", flattenGlueJobCommand(job.Command)); err != nil {
		return fmt.Errorf("error setting command: %s", err)
	}
	if err := d.Set("connections", flattenGlueConnectionsList(job.Connections)); err != nil {
		return fmt.Errorf("error setting connections: %s", err)
	}
	if err := d.Set("default_arguments", aws.StringValueMap(job.DefaultArguments)); err != nil {
		return fmt.Errorf("error setting default_arguments: %s", err)
	}
	d.Set("description", job.Description)
	if err := d.Set("execution_property", flattenGlueExecutionProperty(job.ExecutionProperty)); err != nil {
		return fmt.Errorf("error setting execution_property: %s", err)
	}
	d.Set("max_retries", int(aws.Int64Value(job.MaxRetries)))
	d.Set("name", job.Name)
	d.Set("role_arn", job.Role)
	d.Set("timeout", int(aws.Int64Value(job.Timeout)))

	return nil
}

func resourceAwsGlueJobUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glueconn

	jobUpdate := &glue.JobUpdate{
		Command: expandGlueJobCommand(d.Get("command").([]interface{})),
		Role:    aws.String(d.Get("role_arn").(string)),
		Timeout: aws.Int64(int64(d.Get("timeout").(int))),
	}

	if v, ok := d.GetOk("allocated_capacity"); ok {
		jobUpdate.AllocatedCapacity = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("connections"); ok {
		jobUpdate.Connections = &glue.ConnectionsList{
			Connections: expandStringList(v.([]interface{})),
		}
	}

	if kv, ok := d.GetOk("default_arguments"); ok {
		defaultArgumentsMap := make(map[string]string)
		for k, v := range kv.(map[string]interface{}) {
			defaultArgumentsMap[k] = v.(string)
		}
		jobUpdate.DefaultArguments = aws.StringMap(defaultArgumentsMap)
	}

	if v, ok := d.GetOk("description"); ok {
		jobUpdate.Description = aws.String(v.(string))
	}

	if v, ok := d.GetOk("execution_property"); ok {
		jobUpdate.ExecutionProperty = expandGlueExecutionProperty(v.([]interface{}))
	}

	if v, ok := d.GetOk("max_retries"); ok {
		jobUpdate.MaxRetries = aws.Int64(int64(v.(int)))
	}

	input := &glue.UpdateJobInput{
		JobName:   aws.String(d.Id()),
		JobUpdate: jobUpdate,
	}

	log.Printf("[DEBUG] Updating Glue Job: %s", input)
	_, err := conn.UpdateJob(input)
	if err != nil {
		return fmt.Errorf("error updating Glue Job (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceAwsGlueJobDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).glueconn

	log.Printf("[DEBUG] Deleting Glue Job: %s", d.Id())
	err := deleteGlueJob(conn, d.Id())
	if err != nil {
		return fmt.Errorf("error deleting Glue Job (%s): %s", d.Id(), err)
	}

	return nil
}

func deleteGlueJob(conn *glue.Glue, jobName string) error {
	input := &glue.DeleteJobInput{
		JobName: aws.String(jobName),
	}

	_, err := conn.DeleteJob(input)
	if err != nil {
		if isAWSErr(err, glue.ErrCodeEntityNotFoundException, "") {
			return nil
		}
		return err
	}

	return nil
}

func expandGlueExecutionProperty(l []interface{}) *glue.ExecutionProperty {
	m := l[0].(map[string]interface{})

	executionProperty := &glue.ExecutionProperty{
		MaxConcurrentRuns: aws.Int64(int64(m["max_concurrent_runs"].(int))),
	}

	return executionProperty
}

func expandGlueJobCommand(l []interface{}) *glue.JobCommand {
	m := l[0].(map[string]interface{})

	jobCommand := &glue.JobCommand{
		Name:           aws.String(m["name"].(string)),
		ScriptLocation: aws.String(m["script_location"].(string)),
	}

	return jobCommand
}

func flattenGlueConnectionsList(connectionsList *glue.ConnectionsList) []interface{} {
	if connectionsList == nil {
		return []interface{}{}
	}

	return flattenStringList(connectionsList.Connections)
}

func flattenGlueExecutionProperty(executionProperty *glue.ExecutionProperty) []map[string]interface{} {
	if executionProperty == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"max_concurrent_runs": int(aws.Int64Value(executionProperty.MaxConcurrentRuns)),
	}

	return []map[string]interface{}{m}
}

func flattenGlueJobCommand(jobCommand *glue.JobCommand) []map[string]interface{} {
	if jobCommand == nil {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"name":            aws.StringValue(jobCommand.Name),
		"script_location": aws.StringValue(jobCommand.ScriptLocation),
	}

	return []map[string]interface{}{m}
}
