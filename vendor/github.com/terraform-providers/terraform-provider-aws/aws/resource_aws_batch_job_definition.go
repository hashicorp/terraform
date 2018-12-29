package aws

import (
	"fmt"

	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/structure"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsBatchJobDefinition() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsBatchJobDefinitionCreate,
		Read:   resourceAwsBatchJobDefinitionRead,
		Delete: resourceAwsBatchJobDefinitionDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateBatchName,
			},
			"container_properties": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					json, _ := structure.NormalizeJsonString(v)
					return json
				},
				DiffSuppressFunc: suppressEquivalentJsonDiffs,
				ValidateFunc:     validateAwsBatchJobContainerProperties,
			},
			"parameters": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"retry_strategy": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attempts": {
							Type:         schema.TypeInt,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validation.IntBetween(1, 10),
						},
					},
				},
			},
			"timeout": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attempt_duration_seconds": {
							Type:         schema.TypeInt,
							Optional:     true,
							ForceNew:     true,
							ValidateFunc: validation.IntAtLeast(60),
						},
					},
				},
			},
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{batch.JobDefinitionTypeContainer}, true),
			},
			"revision": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAwsBatchJobDefinitionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).batchconn
	name := d.Get("name").(string)

	input := &batch.RegisterJobDefinitionInput{
		JobDefinitionName: aws.String(name),
		Type:              aws.String(d.Get("type").(string)),
	}

	if v, ok := d.GetOk("container_properties"); ok {
		props, err := expandBatchJobContainerProperties(v.(string))
		if err != nil {
			return fmt.Errorf("%s %q", err, name)
		}
		input.ContainerProperties = props
	}

	if v, ok := d.GetOk("parameters"); ok {
		input.Parameters = expandJobDefinitionParameters(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("retry_strategy"); ok {
		input.RetryStrategy = expandJobDefinitionRetryStrategy(v.([]interface{}))
	}

	if v, ok := d.GetOk("timeout"); ok {
		input.Timeout = expandJobDefinitionTimeout(v.([]interface{}))
	}

	out, err := conn.RegisterJobDefinition(input)
	if err != nil {
		return fmt.Errorf("%s %q", err, name)
	}
	d.SetId(*out.JobDefinitionArn)
	d.Set("arn", out.JobDefinitionArn)
	return resourceAwsBatchJobDefinitionRead(d, meta)
}

func resourceAwsBatchJobDefinitionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).batchconn
	arn := d.Get("arn").(string)
	job, err := getJobDefinition(conn, arn)
	if err != nil {
		return fmt.Errorf("%s %q", err, arn)
	}
	if job == nil {
		d.SetId("")
		return nil
	}
	d.Set("arn", job.JobDefinitionArn)
	d.Set("container_properties", job.ContainerProperties)
	d.Set("parameters", aws.StringValueMap(job.Parameters))

	if err := d.Set("retry_strategy", flattenBatchRetryStrategy(job.RetryStrategy)); err != nil {
		return fmt.Errorf("error setting retry_strategy: %s", err)
	}

	if err := d.Set("timeout", flattenBatchJobTimeout(job.Timeout)); err != nil {
		return fmt.Errorf("error setting timeout: %s", err)
	}

	d.Set("revision", job.Revision)
	d.Set("type", job.Type)
	return nil
}

func resourceAwsBatchJobDefinitionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).batchconn
	arn := d.Get("arn").(string)
	_, err := conn.DeregisterJobDefinition(&batch.DeregisterJobDefinitionInput{
		JobDefinition: aws.String(arn),
	})
	if err != nil {
		return fmt.Errorf("%s %q", err, arn)
	}

	return nil
}

func getJobDefinition(conn *batch.Batch, arn string) (*batch.JobDefinition, error) {
	describeOpts := &batch.DescribeJobDefinitionsInput{
		JobDefinitions: []*string{aws.String(arn)},
	}
	resp, err := conn.DescribeJobDefinitions(describeOpts)
	if err != nil {
		return nil, err
	}

	numJobDefinitions := len(resp.JobDefinitions)
	switch {
	case numJobDefinitions == 0:
		return nil, nil
	case numJobDefinitions == 1:
		if *resp.JobDefinitions[0].Status == "ACTIVE" {
			return resp.JobDefinitions[0], nil
		}
		return nil, nil
	case numJobDefinitions > 1:
		return nil, fmt.Errorf("Multiple Job Definitions with name %s", arn)
	}
	return nil, nil
}

func validateAwsBatchJobContainerProperties(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	_, err := expandBatchJobContainerProperties(value)
	if err != nil {
		errors = append(errors, fmt.Errorf("AWS Batch Job container_properties is invalid: %s", err))
	}
	return
}

func expandBatchJobContainerProperties(rawProps string) (*batch.ContainerProperties, error) {
	var props *batch.ContainerProperties

	err := json.Unmarshal([]byte(rawProps), &props)
	if err != nil {
		return nil, fmt.Errorf("Error decoding JSON: %s", err)
	}

	return props, nil
}

func expandJobDefinitionParameters(params map[string]interface{}) map[string]*string {
	var jobParams = make(map[string]*string)
	for k, v := range params {
		jobParams[k] = aws.String(v.(string))
	}

	return jobParams
}

func expandJobDefinitionRetryStrategy(item []interface{}) *batch.RetryStrategy {
	retryStrategy := &batch.RetryStrategy{}
	data := item[0].(map[string]interface{})

	if v, ok := data["attempts"].(int); ok && v > 0 && v <= 10 {
		retryStrategy.Attempts = aws.Int64(int64(v))
	}

	return retryStrategy
}

func flattenBatchRetryStrategy(item *batch.RetryStrategy) []map[string]interface{} {
	data := []map[string]interface{}{}
	if item != nil && item.Attempts != nil {
		data = append(data, map[string]interface{}{
			"attempts": int(aws.Int64Value(item.Attempts)),
		})
	}
	return data
}

func expandJobDefinitionTimeout(item []interface{}) *batch.JobTimeout {
	timeout := &batch.JobTimeout{}
	data := item[0].(map[string]interface{})

	if v, ok := data["attempt_duration_seconds"].(int); ok && v >= 60 {
		timeout.AttemptDurationSeconds = aws.Int64(int64(v))
	}

	return timeout
}

func flattenBatchJobTimeout(item *batch.JobTimeout) []map[string]interface{} {
	data := []map[string]interface{}{}
	if item != nil && item.AttemptDurationSeconds != nil {
		data = append(data, map[string]interface{}{
			"attempt_duration_seconds": int(aws.Int64Value(item.AttemptDurationSeconds)),
		})
	}
	return data
}
