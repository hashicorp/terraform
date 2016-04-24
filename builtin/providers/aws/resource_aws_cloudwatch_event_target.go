package aws

import (
	"fmt"
	"log"
	"regexp"

	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	events "github.com/aws/aws-sdk-go/service/cloudwatchevents"
)

func resourceAwsCloudWatchEventTarget() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsCloudWatchEventTargetCreate,
		Read:   resourceAwsCloudWatchEventTargetRead,
		Update: resourceAwsCloudWatchEventTargetUpdate,
		Delete: resourceAwsCloudWatchEventTargetDelete,

		Schema: map[string]*schema.Schema{
			"rule": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateCloudWatchEventRuleName,
			},

			"target_id": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ForceNew:     true,
				ValidateFunc: validateCloudWatchEventTargetId,
			},

			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"input": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"input_path"},
				// We could be normalizing the JSON here,
				// but for built-in targets input may not be JSON
			},

			"input_path": &schema.Schema{
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"input"},
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
		}
		return err
	}
	log.Printf("[DEBUG] Found Event Target: %s", t)

	d.Set("arn", t.Arn)
	d.Set("target_id", t.Id)
	d.Set("input", t.Input)
	d.Set("input_path", t.InputPath)

	return nil
}

func findEventTargetById(id, rule string, nextToken *string, conn *events.CloudWatchEvents) (
	*events.Target, error) {
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

	d.SetId("")

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

	input := events.PutTargetsInput{
		Rule:    aws.String(d.Get("rule").(string)),
		Targets: []*events.Target{e},
	}

	return &input
}
