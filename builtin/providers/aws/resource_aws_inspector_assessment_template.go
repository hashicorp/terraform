package aws

import (
	"bytes"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/inspector"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAWSInspectorAssessmentTemplate() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsInspectorAssessmentTemplateCreate,
		Read:   resourceAwsInspectorAssessmentTemplateRead,
		Update: resourceAwsInspectorAssessmentTemplateUpdate,
		Delete: resourceAwsInspectorAssessmentTemplateDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"target_arn": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
				ForceNew: true,
			},
			"duration": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"rules_package_arns": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				Required: true,
				ForceNew: true,
			},
			"subscribe_to_event": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"event": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateSubscribeToEvent,
						},
						"topic_arn": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
				Set: func(v interface{}) int {
					var buf bytes.Buffer
					m := v.(map[string]interface{})
					buf.WriteString(fmt.Sprintf("%s-%s", m["event"].(string), m["topic_arn"].(string)))
					return hashcode.String(buf.String())
				},
			},
		},
	}
}

func resourceAwsInspectorAssessmentTemplateCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	rules := []*string{}
	if attr := d.Get("rules_package_arns").(*schema.Set); attr.Len() > 0 {
		rules = expandStringList(attr.List())
	}

	targetArn := d.Get("target_arn").(string)
	templateName := d.Get("name").(string)
	duration := int64(d.Get("duration").(int))

	resp, err := conn.CreateAssessmentTemplate(&inspector.CreateAssessmentTemplateInput{
		AssessmentTargetArn:    aws.String(targetArn),
		AssessmentTemplateName: aws.String(templateName),
		DurationInSeconds:      aws.Int64(duration),
		RulesPackageArns:       rules,
	})
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] Inspector Assessment Template %s created", *resp.AssessmentTemplateArn)

	d.Set("arn", resp.AssessmentTemplateArn)

	d.SetId(*resp.AssessmentTemplateArn)

	subscriptions := d.Get("subscribe_to_event").(*schema.Set)

	for _, s := range subscriptions.List() {
		m := s.(map[string]interface{})
		event := m["event"].(string)
		topicArn := m["topic_arn"].(string)
		_, err := conn.SubscribeToEvent(&inspector.SubscribeToEventInput{
			Event:       &event,
			TopicArn:    &topicArn,
			ResourceArn: resp.AssessmentTemplateArn,
		})
		if err != nil {
			return err
		}
	}

	return resourceAwsInspectorAssessmentTemplateRead(d, meta)
}

func resourceAwsInspectorAssessmentTemplateRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	resp, err := conn.DescribeAssessmentTemplates(&inspector.DescribeAssessmentTemplatesInput{
		AssessmentTemplateArns: []*string{
			aws.String(d.Id()),
		},
	},
	)
	if err != nil {
		if inspectorerr, ok := err.(awserr.Error); ok && inspectorerr.Code() == "InvalidInputException" {
			return nil
		} else {
			log.Printf("[ERROR] Error finding Inspector Assessment Template: %s", err)
			return err
		}
	}

	if resp.AssessmentTemplates != nil && len(resp.AssessmentTemplates) > 0 {
		d.Set("name", resp.AssessmentTemplates[0].Name)
	}

	ste, err := flattenSubscribeToEvents(d, conn)
	if err != nil {
		return nil
	}
	d.Set("subscribe_to_event", ste)

	return nil
}

func resourceAwsInspectorAssessmentTemplateUpdate(d *schema.ResourceData, meta interface{}) error {
	if d.HasChange("subscribe_to_event") {
		conn := meta.(*AWSClient).inspectorconn

		var new []map[string]interface{}
		var old []map[string]interface{}
		oldSubscribeToEvents, newSubscribeToEvents := d.GetChange("subscribe_to_event")

		for _, o := range oldSubscribeToEvents.(*schema.Set).List() {
			old = append(old, o.(map[string]interface{}))
		}
		for _, n := range newSubscribeToEvents.(*schema.Set).List() {
			new = append(new, n.(map[string]interface{}))
		}

		for _, s := range substractEventSubscriptions(new, old) {
			e := s["event"].(string)
			t := s["topic_arn"].(string)
			r := d.Id()

			_, err := conn.SubscribeToEvent(&inspector.SubscribeToEventInput{
				Event:       &e,
				ResourceArn: &r,
				TopicArn:    &t,
			})
			if err != nil {
				return err
			}
		}

		for _, s := range substractEventSubscriptions(old, new) {
			e := s["event"].(string)
			t := s["topic_arn"].(string)
			r := d.Id()

			_, err := conn.UnsubscribeFromEvent(&inspector.UnsubscribeFromEventInput{
				Event:       &e,
				ResourceArn: &r,
				TopicArn:    &t,
			})
			if err != nil {
				return err
			}
		}

		return resourceAwsInspectorAssessmentTemplateRead(d, meta)
	}
	return nil
}

func resourceAwsInspectorAssessmentTemplateDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).inspectorconn

	// subscriptions to events are removed together with the template automatically

	_, err := conn.DeleteAssessmentTemplate(&inspector.DeleteAssessmentTemplateInput{
		AssessmentTemplateArn: aws.String(d.Id()),
	})
	if err != nil {
		if inspectorerr, ok := err.(awserr.Error); ok && inspectorerr.Code() == "AssessmentRunInProgressException" {
			log.Printf("[ERROR] Assement Run in progress: %s", err)
			return err
		} else {
			log.Printf("[ERROR] Error deleting Assement Template: %s", err)
			return err
		}
	}

	return nil
}

// validateSubscribeToEvent validates the string is a known keyword
func validateSubscribeToEvent(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)

	// Support for "OTHER" is documented but the API returns 400s. See
	// http://docs.aws.amazon.com/inspector/latest/APIReference/API_SubscribeToEvent.html
	switch value {
	case "ASSESSMENT_RUN_STARTED", "ASSESSMENT_RUN_COMPLETED", "ASSESSMENT_RUN_STATE_CHANGED", "FINDING_REPORTED":
		return
	default:
		errors = append(errors, fmt.Errorf("unknown subscription event: %v", v))
	}
	return
}

func flattenSubscribeToEvents(d *schema.ResourceData, conn *inspector.Inspector) ([]map[string]interface{}, error) {
	arn := d.Id()
	var results []map[string]interface{}
	var err error = nil
	var nextToken *string = nil
	var maxResults int64 = 100

	for {
		outPut, err := conn.ListEventSubscriptions(&inspector.ListEventSubscriptionsInput{MaxResults: &maxResults, NextToken: nextToken, ResourceArn: &arn})
		if err != nil {
			return results, err
		}

		for _, s := range outPut.Subscriptions {
			for _, es := range s.EventSubscriptions {
				m := make(map[string]interface{})
				m["event"] = *es.Event
				m["topic_arn"] = *s.TopicArn
				results = append(results, m)
			}
		}

		nextToken = outPut.NextToken
		if nextToken == nil {
			break
		}
	}

	return results, err
}

// substractEventSubscriptions return elements of 'a' which are not contained in 'b'
func substractEventSubscriptions(a []map[string]interface{}, b []map[string]interface{}) (result []map[string]interface{}) {
	for _, e := range a {
		if !containsEventSubscription(b, e) {
			result = append(result, e)
		}
	}
	return
}

func containsEventSubscription(s []map[string]interface{}, e map[string]interface{}) bool {
	for _, a := range s {
		if a["event"] == e["event"] && a["topic_arn"] == e["topic_arn"] {
			return true
		}
	}
	return false
}
