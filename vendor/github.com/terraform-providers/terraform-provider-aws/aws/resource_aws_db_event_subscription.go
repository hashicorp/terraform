package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
		Importer: &schema.ResourceImporter{
			State: resourceAwsDbEventSubscriptionImport,
		},
		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(40 * time.Minute),
			Delete: schema.DefaultTimeout(40 * time.Minute),
			Update: schema.DefaultTimeout(40 * time.Minute),
		},
		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"name": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateDbEventSubscriptionName,
			},
			"sns_topic": {
				Type:     schema.TypeString,
				Required: true,
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
				// ValidateFunc: validateDbEventSubscriptionSourceIds,
				// requires source_type to be set, does not seem to be a way to validate this
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
			"customer_aws_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsDbEventSubscriptionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
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

	output, err := conn.CreateEventSubscription(request)
	if err != nil || output.EventSubscription == nil {
		return fmt.Errorf("Error creating RDS Event Subscription %s: %s", name, err)
	}

	d.SetId(aws.StringValue(output.EventSubscription.CustSubscriptionId))

	if err := setTagsRDS(conn, d, aws.StringValue(output.EventSubscription.EventSubscriptionArn)); err != nil {
		return fmt.Errorf("Error creating RDS Event Subscription (%s) tags: %s", d.Id(), err)
	}

	log.Println(
		"[INFO] Waiting for RDS Event Subscription to be ready")

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating"},
		Target:     []string{"active"},
		Refresh:    resourceAwsDbEventSubscriptionRefreshFunc(d.Id(), conn),
		Timeout:    d.Timeout(schema.TimeoutCreate),
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
	conn := meta.(*AWSClient).rdsconn

	sub, err := resourceAwsDbEventSubscriptionRetrieve(d.Get("name").(string), conn)
	if err != nil {
		return fmt.Errorf("Error retrieving RDS Event Subscription %s: %s", d.Id(), err)
	}
	if sub == nil {
		log.Printf("[WARN] RDS Event Subscription (%s) not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("arn", sub.EventSubscriptionArn)
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
	resp, err := conn.ListTagsForResource(&rds.ListTagsForResourceInput{
		ResourceName: sub.EventSubscriptionArn,
	})

	if err != nil {
		log.Printf("[DEBUG] Error retrieving tags for ARN: %s", aws.StringValue(sub.EventSubscriptionArn))
	}

	var dt []*rds.Tag
	if len(resp.TagList) > 0 {
		dt = resp.TagList
	}
	if err := d.Set("tags", tagsToMapRDS(dt)); err != nil {
		return fmt.Errorf("error setting tags: %s", err)
	}

	return nil
}

func resourceAwsDbEventSubscriptionRetrieve(name string, conn *rds.RDS) (*rds.EventSubscription, error) {

	request := &rds.DescribeEventSubscriptionsInput{
		SubscriptionName: aws.String(name),
	}

	describeResp, err := conn.DescribeEventSubscriptions(request)
	if err != nil {
		if isAWSErr(err, rds.ErrCodeSubscriptionNotFoundFault, "") {
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
	conn := meta.(*AWSClient).rdsconn

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
		req.SourceType = aws.String(d.Get("source_type").(string))
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
		_, err := conn.ModifyEventSubscription(req)
		if err != nil {
			return fmt.Errorf("Modifying RDS Event Subscription %s failed: %s", d.Id(), err)
		}

		log.Println(
			"[INFO] Waiting for RDS Event Subscription modification to finish")

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"modifying"},
			Target:     []string{"active"},
			Refresh:    resourceAwsDbEventSubscriptionRefreshFunc(d.Id(), conn),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
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

	if err := setTagsRDS(conn, d, d.Get("arn").(string)); err != nil {
		return err
	} else {
		d.SetPartial("tags")
	}

	if d.HasChange("source_ids") {
		o, n := d.GetChange("source_ids")
		if o == nil {
			o = new(schema.Set)
		}
		if n == nil {
			n = new(schema.Set)
		}

		os := o.(*schema.Set)
		ns := n.(*schema.Set)
		remove := expandStringList(os.Difference(ns).List())
		add := expandStringList(ns.Difference(os).List())

		if len(remove) > 0 {
			for _, removing := range remove {
				log.Printf("[INFO] Removing %s as a Source Identifier from %q", *removing, d.Id())
				_, err := conn.RemoveSourceIdentifierFromSubscription(&rds.RemoveSourceIdentifierFromSubscriptionInput{
					SourceIdentifier: removing,
					SubscriptionName: aws.String(d.Id()),
				})
				if err != nil {
					return err
				}
			}
		}

		if len(add) > 0 {
			for _, adding := range add {
				log.Printf("[INFO] Adding %s as a Source Identifier to %q", *adding, d.Id())
				_, err := conn.AddSourceIdentifierToSubscription(&rds.AddSourceIdentifierToSubscriptionInput{
					SourceIdentifier: adding,
					SubscriptionName: aws.String(d.Id()),
				})
				if err != nil {
					return err
				}
			}
		}
		d.SetPartial("source_ids")
	}

	d.Partial(false)

	return nil
}

func resourceAwsDbEventSubscriptionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	deleteOpts := rds.DeleteEventSubscriptionInput{
		SubscriptionName: aws.String(d.Id()),
	}

	if _, err := conn.DeleteEventSubscription(&deleteOpts); err != nil {
		if isAWSErr(err, rds.ErrCodeSubscriptionNotFoundFault, "") {
			return nil
		}
		return fmt.Errorf("Error deleting RDS Event Subscription %s: %s", d.Id(), err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting"},
		Target:     []string{},
		Refresh:    resourceAwsDbEventSubscriptionRefreshFunc(d.Id(), conn),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second, // Wait 30 secs before starting
	}
	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting RDS Event Subscription %s: %s", d.Id(), err)
	}
	return err
}

func resourceAwsDbEventSubscriptionRefreshFunc(name string, conn *rds.RDS) resource.StateRefreshFunc {

	return func() (interface{}, string, error) {
		sub, err := resourceAwsDbEventSubscriptionRetrieve(name, conn)

		if err != nil {
			log.Printf("Error on retrieving DB Event Subscription when waiting: %s", err)
			return nil, "", err
		}

		if sub == nil {
			return nil, "", nil
		}

		if sub.Status != nil {
			log.Printf("[DEBUG] DB Event Subscription status for %s: %s", name, *sub.Status)
		}

		return sub, *sub.Status, nil
	}
}
