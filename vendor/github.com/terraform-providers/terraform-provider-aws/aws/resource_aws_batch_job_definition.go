package aws

import (
	"fmt"

	"encoding/json"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/hashicorp/terraform/helper/schema"
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
					json, _ := normalizeJsonString(v)
					return json
				},
				DiffSuppressFunc: suppressEquivalentJsonDiffs,
				ValidateFunc:     validateAwsBatchJobContainerProperties,
			},
			"parameters": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
				Elem:     schema.TypeString,
			},
			"retry_strategy": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"attempts": {
							Type:     schema.TypeInt,
							Required: true,
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
	d.Set("retry_strategy", flattenRetryStrategy(job.RetryStrategy))
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
	d.SetId("")
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
	data := item[0].(map[string]interface{})
	return &batch.RetryStrategy{
		Attempts: aws.Int64(int64(data["attempts"].(int))),
	}
}

func flattenRetryStrategy(item *batch.RetryStrategy) []map[string]interface{} {
	data := []map[string]interface{}{}
	if item != nil {
		data = append(data, map[string]interface{}{
			"attempts": item.Attempts,
		})
	}
	return data
}
