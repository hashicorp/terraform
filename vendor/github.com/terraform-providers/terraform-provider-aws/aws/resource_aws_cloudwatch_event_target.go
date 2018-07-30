package aws

import (
	"fmt"
	"log"
	"math"
	"regexp"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	events "github.com/aws/aws-sdk-go/service/cloudwatchevents"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsCloudWatchEventTarget() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudWatchEventTargetCreate,
		Read:   resourceAwsCloudWatchEventTargetRead,
		Update: resourceAwsCloudWatchEventTargetUpdate,
		Delete: resourceAwsCloudWatchEventTargetDelete,

		Schema: map[string]*schema.Schema{
			"rule": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateCloudWatchEventRuleName,
			},

			"target_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateCloudWatchEventTargetId,
			},

			"arn": {
				Type:     schema.TypeString,
				Required: true,
			},

			"input": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"input_path"},
				// We could be normalizing the JSON here,
				// but for built-in targets input may not be JSON
			},

			"input_path": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"input"},
			},

			"role_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},

			"run_command_targets": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 5,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(1, 128),
						},
						"values": {
							Type:     schema.TypeList,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},

			"ecs_target": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"task_count": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(1, math.MaxInt32),
						},
						"task_definition_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(1, 1600),
						},
					},
				},
			},

			"batch_target": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"job_definition": {
							Type:     schema.TypeString,
							Required: true,
						},
						"job_name": {
							Type:     schema.TypeString,
							Required: true,
						},
						"array_size": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(2, 10000),
						},
						"job_attempts": {
							Type:         schema.TypeInt,
							Optional:     true,
							ValidateFunc: validation.IntBetween(1, 10),
						},
					},
				},
			},

			"kinesis_target": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"partition_key_path": {
							Type:         schema.TypeString,
							Optional:     true,
							ValidateFunc: validation.StringLenBetween(1, 256),
						},
					},
				},
			},

			"sqs_target": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"message_group_id": {
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},

			"input_transformer": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"input_paths": {
							Type:     schema.TypeMap,
							Optional: true,
						},
						"input_template": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validation.StringLenBetween(1, 8192),
						},
					},
				},
			},
		},
	}
}

func resourceAwsCloudWatchEventTargetCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatcheventsconn

	rule := d.Get("rule").(string)

	var targetId string
	if v, ok := d.GetOk("target_id"); ok {
		targetId = v.(string)
	} else {
		targetId = resource.UniqueId()
		d.Set("target_id", targetId)
	}

	input := buildPutTargetInputStruct(d)

	log.Printf("[DEBUG] Creating CloudWatch Event Target: %s", input)
	out, err := conn.PutTargets(input)
	if err != nil {
		return fmt.Errorf("Creating CloudWatch Event Target failed: %s", err)
	}

	if len(out.FailedEntries) > 0 {
		return fmt.Errorf("Creating CloudWatch Event Target failed: %s",
			out.FailedEntries)
	}

	id := rule + "-" + targetId
	d.SetId(id)

	log.Printf("[INFO] CloudWatch Event Target %q created", d.Id())

	return resourceAwsCloudWatchEventTargetRead(d, meta)
}

func resourceAwsCloudWatchEventTargetRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatcheventsconn

	t, err := findEventTargetById(
		d.Get("target_id").(string),
		d.Get("rule").(string),
		nil, conn)
	if err != nil {
		if regexp.MustCompile(" not found$").MatchString(err.Error()) {
			log.Printf("[WARN] Removing CloudWatch Event Target %q because it's gone.", d.Id())
			d.SetId("")
			return nil
		}
		if awsErr, ok := err.(awserr.Error); ok {
			// This should never happen, but it's useful
			// for recovering from https://github.com/hashicorp/terraform/issues/5389
			if awsErr.Code() == "ValidationException" {
				log.Printf("[WARN] Removing CloudWatch Event Target %q because it never existed.", d.Id())
				d.SetId("")
				return nil
			}

			if awsErr.Code() == "ResourceNotFoundException" {
				log.Printf("[WARN] CloudWatch Event Target (%q) not found. Removing it from state.", d.Id())
				d.SetId("")
				return nil
			}

		}
		return err
	}
	log.Printf("[DEBUG] Found Event Target: %s", t)

	d.Set("arn", t.Arn)
	d.Set("target_id", t.Id)
	d.Set("input", t.Input)
	d.Set("input_path", t.InputPath)
	d.Set("role_arn", t.RoleArn)

	if t.RunCommandParameters != nil {
		if err := d.Set("run_command_targets", flattenAwsCloudWatchEventTargetRunParameters(t.RunCommandParameters)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting run_command_targets error: %#v", err)
		}
	}

	if t.EcsParameters != nil {
		if err := d.Set("ecs_target", flattenAwsCloudWatchEventTargetEcsParameters(t.EcsParameters)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting ecs_target error: %#v", err)
		}
	}

	if t.BatchParameters != nil {
		if err := d.Set("batch_target", flattenAwsCloudWatchEventTargetBatchParameters(t.BatchParameters)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting batch_target error: %#v", err)
		}
	}

	if t.KinesisParameters != nil {
		if err := d.Set("kinesis_target", flattenAwsCloudWatchEventTargetKinesisParameters(t.KinesisParameters)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting kinesis_target error: %#v", err)
		}
	}

	if t.SqsParameters != nil {
		if err := d.Set("sqs_target", flattenAwsCloudWatchEventTargetSqsParameters(t.SqsParameters)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting sqs_target error: %#v", err)
		}
	}

	if t.InputTransformer != nil {
		if err := d.Set("input_transformer", flattenAwsCloudWatchInputTransformer(t.InputTransformer)); err != nil {
			return fmt.Errorf("[DEBUG] Error setting input_transformer error: %#v", err)
		}
	}

	return nil
}

func findEventTargetById(id, rule string, nextToken *string, conn *events.CloudWatchEvents) (*events.Target, error) {
	input := events.ListTargetsByRuleInput{
		Rule:      aws.String(rule),
		NextToken: nextToken,
		Limit:     aws.Int64(100), // Set limit to allowed maximum to prevent API throttling
	}
	log.Printf("[DEBUG] Reading CloudWatch Event Target: %s", input)
	out, err := conn.ListTargetsByRule(&input)
	if err != nil {
		return nil, err
	}

	for _, t := range out.Targets {
		if *t.Id == id {
			return t, nil
		}
	}

	if out.NextToken != nil {
		return findEventTargetById(id, rule, nextToken, conn)
	}

	return nil, fmt.Errorf("CloudWatch Event Target %q (%q) not found", id, rule)
}

func resourceAwsCloudWatchEventTargetUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatcheventsconn

	input := buildPutTargetInputStruct(d)

	log.Printf("[DEBUG] Updating CloudWatch Event Target: %s", input)
	_, err := conn.PutTargets(input)
	if err != nil {
		return fmt.Errorf("Updating CloudWatch Event Target failed: %s", err)
	}

	return resourceAwsCloudWatchEventTargetRead(d, meta)
}

func resourceAwsCloudWatchEventTargetDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).cloudwatcheventsconn

	input := events.RemoveTargetsInput{
		Ids:  []*string{aws.String(d.Get("target_id").(string))},
		Rule: aws.String(d.Get("rule").(string)),
	}
	log.Printf("[INFO] Deleting CloudWatch Event Target: %s", input)
	_, err := conn.RemoveTargets(&input)
	if err != nil {
		return fmt.Errorf("Error deleting CloudWatch Event Target: %s", err)
	}
	log.Println("[INFO] CloudWatch Event Target deleted")

	return nil
}

func buildPutTargetInputStruct(d *schema.ResourceData) *events.PutTargetsInput {
	e := &events.Target{
		Arn: aws.String(d.Get("arn").(string)),
		Id:  aws.String(d.Get("target_id").(string)),
	}

	if v, ok := d.GetOk("input"); ok {
		e.Input = aws.String(v.(string))
	}
	if v, ok := d.GetOk("input_path"); ok {
		e.InputPath = aws.String(v.(string))
	}

	if v, ok := d.GetOk("role_arn"); ok {
		e.RoleArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("run_command_targets"); ok {
		e.RunCommandParameters = expandAwsCloudWatchEventTargetRunParameters(v.([]interface{}))
	}
	if v, ok := d.GetOk("ecs_target"); ok {
		e.EcsParameters = expandAwsCloudWatchEventTargetEcsParameters(v.([]interface{}))
	}
	if v, ok := d.GetOk("batch_target"); ok {
		e.BatchParameters = expandAwsCloudWatchEventTargetBatchParameters(v.([]interface{}))
	}

	if v, ok := d.GetOk("kinesis_target"); ok {
		e.KinesisParameters = expandAwsCloudWatchEventTargetKinesisParameters(v.([]interface{}))
	}

	if v, ok := d.GetOk("sqs_target"); ok {
		e.SqsParameters = expandAwsCloudWatchEventTargetSqsParameters(v.([]interface{}))
	}

	if v, ok := d.GetOk("input_transformer"); ok {
		e.InputTransformer = expandAwsCloudWatchEventTransformerParameters(v.([]interface{}))
	}

	input := events.PutTargetsInput{
		Rule:    aws.String(d.Get("rule").(string)),
		Targets: []*events.Target{e},
	}

	return &input
}

func expandAwsCloudWatchEventTargetRunParameters(config []interface{}) *events.RunCommandParameters {

	commands := make([]*events.RunCommandTarget, 0)

	for _, c := range config {
		param := c.(map[string]interface{})
		command := &events.RunCommandTarget{
			Key:    aws.String(param["key"].(string)),
			Values: expandStringList(param["values"].([]interface{})),
		}

		commands = append(commands, command)
	}

	command := &events.RunCommandParameters{
		RunCommandTargets: commands,
	}

	return command
}

func expandAwsCloudWatchEventTargetEcsParameters(config []interface{}) *events.EcsParameters {
	ecsParameters := &events.EcsParameters{}
	for _, c := range config {
		param := c.(map[string]interface{})
		ecsParameters.TaskCount = aws.Int64(int64(param["task_count"].(int)))
		ecsParameters.TaskDefinitionArn = aws.String(param["task_definition_arn"].(string))
	}

	return ecsParameters
}

func expandAwsCloudWatchEventTargetBatchParameters(config []interface{}) *events.BatchParameters {
	batchParameters := &events.BatchParameters{}
	for _, c := range config {
		param := c.(map[string]interface{})
		batchParameters.JobDefinition = aws.String(param["job_definition"].(string))
		batchParameters.JobName = aws.String(param["job_name"].(string))
		if v, ok := param["array_size"].(int); ok && v > 1 && v <= 10000 {
			arrayProperties := &events.BatchArrayProperties{}
			arrayProperties.Size = aws.Int64(int64(v))
			batchParameters.ArrayProperties = arrayProperties
		}
		if v, ok := param["job_attempts"].(int); ok && v > 0 && v <= 10 {
			retryStrategy := &events.BatchRetryStrategy{}
			retryStrategy.Attempts = aws.Int64(int64(v))
			batchParameters.RetryStrategy = retryStrategy
		}
	}

	return batchParameters
}

func expandAwsCloudWatchEventTargetKinesisParameters(config []interface{}) *events.KinesisParameters {
	kinesisParameters := &events.KinesisParameters{}
	for _, c := range config {
		param := c.(map[string]interface{})
		if v, ok := param["partition_key_path"].(string); ok && v != "" {
			kinesisParameters.PartitionKeyPath = aws.String(v)
		}
	}

	return kinesisParameters
}

func expandAwsCloudWatchEventTargetSqsParameters(config []interface{}) *events.SqsParameters {
	sqsParameters := &events.SqsParameters{}
	for _, c := range config {
		param := c.(map[string]interface{})
		if v, ok := param["message_group_id"].(string); ok && v != "" {
			sqsParameters.MessageGroupId = aws.String(v)
		}
	}

	return sqsParameters
}

func expandAwsCloudWatchEventTransformerParameters(config []interface{}) *events.InputTransformer {
	transformerParameters := &events.InputTransformer{}

	inputPathsMaps := map[string]*string{}

	for _, c := range config {
		param := c.(map[string]interface{})
		inputPaths := param["input_paths"].(map[string]interface{})

		for k, v := range inputPaths {
			inputPathsMaps[k] = aws.String(v.(string))
		}
		transformerParameters.InputTemplate = aws.String(param["input_template"].(string))
	}
	transformerParameters.InputPathsMap = inputPathsMaps

	return transformerParameters
}

func flattenAwsCloudWatchEventTargetRunParameters(runCommand *events.RunCommandParameters) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)

	for _, x := range runCommand.RunCommandTargets {
		config := make(map[string]interface{})

		config["key"] = *x.Key
		config["values"] = flattenStringList(x.Values)

		result = append(result, config)
	}

	return result
}
func flattenAwsCloudWatchEventTargetEcsParameters(ecsParameters *events.EcsParameters) []map[string]interface{} {
	config := make(map[string]interface{})
	config["task_count"] = *ecsParameters.TaskCount
	config["task_definition_arn"] = *ecsParameters.TaskDefinitionArn
	result := []map[string]interface{}{config}
	return result
}

func flattenAwsCloudWatchEventTargetBatchParameters(batchParameters *events.BatchParameters) []map[string]interface{} {
	config := make(map[string]interface{})
	config["job_definition"] = aws.StringValue(batchParameters.JobDefinition)
	config["job_name"] = aws.StringValue(batchParameters.JobName)
	if batchParameters.ArrayProperties != nil {
		config["array_size"] = int(aws.Int64Value(batchParameters.ArrayProperties.Size))
	}
	if batchParameters.RetryStrategy != nil {
		config["job_attempts"] = int(aws.Int64Value(batchParameters.RetryStrategy.Attempts))
	}
	result := []map[string]interface{}{config}
	return result
}

func flattenAwsCloudWatchEventTargetKinesisParameters(kinesisParameters *events.KinesisParameters) []map[string]interface{} {
	config := make(map[string]interface{})
	config["partition_key_path"] = *kinesisParameters.PartitionKeyPath
	result := []map[string]interface{}{config}
	return result
}

func flattenAwsCloudWatchEventTargetSqsParameters(sqsParameters *events.SqsParameters) []map[string]interface{} {
	config := make(map[string]interface{})
	config["message_group_id"] = *sqsParameters.MessageGroupId
	result := []map[string]interface{}{config}
	return result
}

func flattenAwsCloudWatchInputTransformer(inputTransformer *events.InputTransformer) []map[string]interface{} {
	config := make(map[string]interface{})
	inputPathsMap := make(map[string]string)
	for k, v := range inputTransformer.InputPathsMap {
		inputPathsMap[k] = *v
	}
	config["input_template"] = *inputTransformer.InputTemplate
	config["input_paths"] = inputPathsMap

	result := []map[string]interface{}{config}
	return result
}
