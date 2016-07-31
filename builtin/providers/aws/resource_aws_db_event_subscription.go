package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsDbEventSubscription() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsDbEventSubscriptionCreate,
		Read:   resourceAwsDbEventSubscriptionRead,
		Update: resourceAwsDbEventSubscriptionUpdate,
		Delete: resourceAwsDbEventSubscriptionDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateDbEventSubscriptionName,
			},
			"sns_topic": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"event_categories": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"source_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
				// ValidateFunc: validateDbEventSubscriptionSourceIds,
				// requires source_type to be set, does not seem to be a way to validate this
			},
			"source_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"customer_aws_id": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDbEventSubscriptionCreate(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn
	name := d.Get("name").(string)
	tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))

	sourceIdsSet := d.Get("source_ids").(*schema.Set)
	sourceIds := make([]*string, sourceIdsSet.Len())
	for i, sourceId := range sourceIdsSet.List() {
		sourceIds[i] = aws.String(sourceId.(string))
	}

	eventCategoriesSet := d.Get("event_categories").(*schema.Set)
	eventCategories := make([]*string, eventCategoriesSet.Len())
	for i, eventCategory := range eventCategoriesSet.List() {
		eventCategories[i] = aws.String(eventCategory.(string))
	}

	request := &rds.CreateEventSubscriptionInput{
		SubscriptionName: aws.String(name),
		SnsTopicArn:      aws.String(d.Get("sns_topic").(string)),
		Enabled:          aws.Bool(d.Get("enabled").(bool)),
		SourceIds:        sourceIds,
		SourceType:       aws.String(d.Get("source_type").(string)),
		EventCategories:  eventCategories,
		Tags:             tags,
	}

	log.Println("[DEBUG] Create RDS Event Subscription:", request)

	_, err := rdsconn.CreateEventSubscription(request)
	if err != nil {
		return fmt.Errorf("Error creating RDS Event Subscription %s: %s", name, err)
	}

	log.Println(
		"[INFO] Waiting for RDS Event Subscription to be ready")

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating"},
		Target:     []string{"active"},
		Refresh:    resourceAwsDbEventSubscriptionRefreshFunc(d, meta.(*AWSClient).rdsconn),
		Timeout:    40 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Creating RDS Event Subscription %s failed: %s", d.Id(), err)
	}

	return resourceAwsDbEventSubscriptionRead(d, meta)
}

func resourceAwsDbEventSubscriptionRead(d *schema.ResourceData, meta interface{}) error {
	sub, err := resourceAwsDbEventSubscriptionRetrieve(d.Get("name").(string), meta.(*AWSClient).rdsconn)
	if err != nil {
		return fmt.Errorf("Error retrieving RDS Event Subscription %s: %s", d.Id(), err)
	}
	if sub == nil {
		d.SetId("")
		return nil
	}

	d.SetId(*sub.CustSubscriptionId)
	if err := d.Set("name", sub.CustSubscriptionId); err != nil {
		return err
	}
	if err := d.Set("sns_topic", sub.SnsTopicArn); err != nil {
		return err
	}
	if err := d.Set("source_type", sub.SourceType); err != nil {
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

	// list tags for resource
	// set tags
	conn := meta.(*AWSClient).rdsconn
	arn := buildRDSEventSubscriptionARN(d.Get("customer_aws_id").(string), d.Id(), meta.(*AWSClient).region)
	resp, err := conn.ListTagsForResource(&rds.ListTagsForResourceInput{
		ResourceName: aws.String(arn),
	})

	if err != nil {
		log.Printf("[DEBUG] Error retrieving tags for ARN: %s", arn)
	}

	var dt []*rds.Tag
	if len(resp.TagList) > 0 {
		dt = resp.TagList
	}
	d.Set("tags", tagsToMapRDS(dt))

	return nil
}

func resourceAwsDbEventSubscriptionRetrieve(
	name string, rdsconn *rds.RDS) (*rds.EventSubscription, error) {

	request := &rds.DescribeEventSubscriptionsInput{
		SubscriptionName: aws.String(name),
	}

	describeResp, err := rdsconn.DescribeEventSubscriptions(request)
	if err != nil {
		if rdserr, ok := err.(awserr.Error); ok && rdserr.Code() == "SubscriptionNotFound" {
			log.Printf("[WARN] No RDS Event Subscription by name (%s) found", name)
			return nil, nil
		}
		return nil, fmt.Errorf("Error reading RDS Event Subscription %s: %s", name, err)
	}

	if len(describeResp.EventSubscriptionsList) != 1 {
		return nil, fmt.Errorf("Unable to find RDS Event Subscription: %#v", describeResp.EventSubscriptionsList)
	}

	return describeResp.EventSubscriptionsList[0], nil
}

func resourceAwsDbEventSubscriptionUpdate(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn

	d.Partial(true)
	requestUpdate := false

	req := &rds.ModifyEventSubscriptionInput{
		SubscriptionName: aws.String(d.Id()),
	}

	if d.HasChange("event_categories") {
		eventCategoriesSet := d.Get("event_categories").(*schema.Set)
		req.EventCategories = make([]*string, eventCategoriesSet.Len())
		for i, eventCategory := range eventCategoriesSet.List() {
			req.EventCategories[i] = aws.String(eventCategory.(string))
		}
		requestUpdate = true
	}

	if d.HasChange("enabled") {
		req.Enabled = aws.Bool(d.Get("enabled").(bool))
		requestUpdate = true
	}

	if d.HasChange("sns_topic") {
		req.SnsTopicArn = aws.String(d.Get("sns_topic").(string))
		requestUpdate = true
	}

	if d.HasChange("source_type") {
		req.SourceType = aws.String(d.Get("source_type").(string))
		requestUpdate = true
	}

	log.Printf("[DEBUG] Send RDS Event Subscription modification request: %#v", requestUpdate)
	if requestUpdate {
		log.Printf("[DEBUG] RDS Event Subscription modification request: %#v", req)
		_, err := rdsconn.ModifyEventSubscription(req)
		if err != nil {
			return fmt.Errorf("Modifying RDS Event Subscription %s failed: %s", d.Id(), err)
		}

		log.Println(
			"[INFO] Waiting for RDS Event Subscription modification to finish")

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"modifying"},
			Target:     []string{"active"},
			Refresh:    resourceAwsDbEventSubscriptionRefreshFunc(d, meta.(*AWSClient).rdsconn),
			Timeout:    40 * time.Minute,
			MinTimeout: 10 * time.Second,
			Delay:      30 * time.Second, // Wait 30 secs before starting
		}

		// Wait, catching any errors
		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Modifying RDS Event Subscription %s failed: %s", d.Id(), err)
		}
		d.SetPartial("event_categories")
		d.SetPartial("enabled")
		d.SetPartial("sns_topic")
		d.SetPartial("source_type")
	}

	arn := buildRDSEventSubscriptionARN(d.Get("customer_aws_id").(string), d.Id(), meta.(*AWSClient).region)
	if err := setTagsRDS(rdsconn, d, arn); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}
	d.Partial(false)

	return nil
}

func resourceAwsDbEventSubscriptionDelete(d *schema.ResourceData, meta interface{}) error {
	rdsconn := meta.(*AWSClient).rdsconn
	deleteOpts := rds.DeleteEventSubscriptionInput{
		SubscriptionName: aws.String(d.Id()),
	}

	if _, err := rdsconn.DeleteEventSubscription(&deleteOpts); err != nil {
		rdserr, ok := err.(awserr.Error)
		if !ok {
			return fmt.Errorf("Error deleting RDS Event Subscription %s: %s", d.Id(), err)
		}

		if rdserr.Code() != "DBEventSubscriptionNotFoundFault" {
			log.Printf("[WARN] RDS Event Subscription %s missing during delete", d.Id())
			return fmt.Errorf("Error deleting RDS Event Subscription %s: %s", d.Id(), err)
		}
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting"},
		Target:     []string{},
		Refresh:    resourceAwsDbEventSubscriptionRefreshFunc(d, meta.(*AWSClient).rdsconn),
		Timeout:    40 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}
	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting RDS Event Subscription %s: %s", d.Id(), err)
	}
	return err
}

func resourceAwsDbEventSubscriptionRefreshFunc(
	d *schema.ResourceData,
	rdsconn *rds.RDS) resource.StateRefreshFunc {

	return func() (interface{}, string, error) {
		sub, err := resourceAwsDbEventSubscriptionRetrieve(d.Get("name").(string), rdsconn)

		if err != nil {
			log.Printf("Error on retrieving DB Event Subscription when waiting: %s", err)
			return nil, "", err
		}

		if sub == nil {
			return nil, "", nil
		}

		if sub.Status != nil {
			log.Printf("[DEBUG] DB Event Subscription status for %s: %s", d.Id(), *sub.Status)
		}

		return sub, *sub.Status, nil
	}
}

func buildRDSEventSubscriptionARN(customerAwsId, subscriptionId, region string) string {
	arn := fmt.Sprintf("arn:aws:rds:%s:%s:es:%s", region, customerAwsId, subscriptionId)
	return arn
}
