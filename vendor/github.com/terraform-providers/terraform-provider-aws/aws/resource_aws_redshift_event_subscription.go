package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsRedshiftEventSubscription() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRedshiftEventSubscriptionCreate,
		Read:   resourceAwsRedshiftEventSubscriptionRead,
		Update: resourceAwsRedshiftEventSubscriptionUpdate,
		Delete: resourceAwsRedshiftEventSubscriptionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(40 * time.Minute),
			Delete: schema.DefaultTimeout(40 * time.Minute),
			Update: schema.DefaultTimeout(40 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"sns_topic_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArn,
			},
			"status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"event_categories": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"source_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"source_type": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"severity": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"customer_aws_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsRedshiftEventSubscriptionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	request := &redshift.CreateEventSubscriptionInput{
		SubscriptionName: aws.String(d.Get("name").(string)),
		SnsTopicArn:      aws.String(d.Get("sns_topic_arn").(string)),
		Enabled:          aws.Bool(d.Get("enabled").(bool)),
		SourceIds:        expandStringSet(d.Get("source_ids").(*schema.Set)),
		SourceType:       aws.String(d.Get("source_type").(string)),
		Severity:         aws.String(d.Get("severity").(string)),
		EventCategories:  expandStringSet(d.Get("event_categories").(*schema.Set)),
		Tags:             tagsFromMapRedshift(d.Get("tags").(map[string]interface{})),
	}

	log.Println("[DEBUG] Create Redshift Event Subscription:", request)

	output, err := conn.CreateEventSubscription(request)
	if err != nil || output.EventSubscription == nil {
		return fmt.Errorf("Error creating Redshift Event Subscription %s: %s", d.Get("name").(string), err)
	}

	d.SetId(aws.StringValue(output.EventSubscription.CustSubscriptionId))

	return resourceAwsRedshiftEventSubscriptionRead(d, meta)
}

func resourceAwsRedshiftEventSubscriptionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	sub, err := resourceAwsRedshiftEventSubscriptionRetrieve(d.Id(), conn)
	if err != nil {
		return fmt.Errorf("Error retrieving Redshift Event Subscription %s: %s", d.Id(), err)
	}
	if sub == nil {
		log.Printf("[WARN] Redshift Event Subscription (%s) not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err := d.Set("name", sub.CustSubscriptionId); err != nil {
		return err
	}
	if err := d.Set("sns_topic_arn", sub.SnsTopicArn); err != nil {
		return err
	}
	if err := d.Set("status", sub.Status); err != nil {
		return err
	}
	if err := d.Set("source_type", sub.SourceType); err != nil {
		return err
	}
	if err := d.Set("severity", sub.Severity); err != nil {
		return err
	}
	if err := d.Set("enabled", sub.Enabled); err != nil {
		return err
	}
	if err := d.Set("source_ids", flattenStringList(sub.SourceIdsList)); err != nil {
		return err
	}
	if err := d.Set("event_categories", flattenStringList(sub.EventCategoriesList)); err != nil {
		return err
	}
	if err := d.Set("customer_aws_id", sub.CustomerAwsId); err != nil {
		return err
	}
	if err := d.Set("tags", tagsToMapRedshift(sub.Tags)); err != nil {
		return err
	}

	return nil
}

func resourceAwsRedshiftEventSubscriptionRetrieve(name string, conn *redshift.Redshift) (*redshift.EventSubscription, error) {

	request := &redshift.DescribeEventSubscriptionsInput{
		SubscriptionName: aws.String(name),
	}

	describeResp, err := conn.DescribeEventSubscriptions(request)
	if err != nil {
		if isAWSErr(err, redshift.ErrCodeSubscriptionNotFoundFault, "") {
			log.Printf("[WARN] No Redshift Event Subscription by name (%s) found", name)
			return nil, nil
		}
		return nil, fmt.Errorf("Error reading Redshift Event Subscription %s: %s", name, err)
	}

	if len(describeResp.EventSubscriptionsList) != 1 {
		return nil, fmt.Errorf("Unable to find Redshift Event Subscription: %#v", describeResp.EventSubscriptionsList)
	}

	return describeResp.EventSubscriptionsList[0], nil
}

func resourceAwsRedshiftEventSubscriptionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	req := &redshift.ModifyEventSubscriptionInput{
		SubscriptionName: aws.String(d.Id()),
		SnsTopicArn:      aws.String(d.Get("sns_topic_arn").(string)),
		Enabled:          aws.Bool(d.Get("enabled").(bool)),
		SourceIds:        expandStringSet(d.Get("source_ids").(*schema.Set)),
		SourceType:       aws.String(d.Get("source_type").(string)),
		Severity:         aws.String(d.Get("severity").(string)),
		EventCategories:  expandStringSet(d.Get("event_categories").(*schema.Set)),
	}

	log.Printf("[DEBUG] Redshift Event Subscription modification request: %#v", req)
	_, err := conn.ModifyEventSubscription(req)
	if err != nil {
		return fmt.Errorf("Modifying Redshift Event Subscription %s failed: %s", d.Id(), err)
	}

	return nil
}

func resourceAwsRedshiftEventSubscriptionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn
	deleteOpts := redshift.DeleteEventSubscriptionInput{
		SubscriptionName: aws.String(d.Id()),
	}

	if _, err := conn.DeleteEventSubscription(&deleteOpts); err != nil {
		if isAWSErr(err, redshift.ErrCodeSubscriptionNotFoundFault, "") {
			return nil
		}
		return fmt.Errorf("Error deleting Redshift Event Subscription %s: %s", d.Id(), err)
	}

	return nil
}
