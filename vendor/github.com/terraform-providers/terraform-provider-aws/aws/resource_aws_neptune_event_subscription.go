package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/neptune"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsNeptuneEventSubscription() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsNeptuneEventSubscriptionCreate,
		Read:   resourceAwsNeptuneEventSubscriptionRead,
		Update: resourceAwsNeptuneEventSubscriptionUpdate,
		Delete: resourceAwsNeptuneEventSubscriptionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
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
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name_prefix"},
				ValidateFunc:  validateNeptuneEventSubscriptionName,
			},
			"name_prefix": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ForceNew:      true,
				ConflictsWith: []string{"name"},
				ValidateFunc:  validateNeptuneEventSubscriptionNamePrefix,
			},
			"sns_topic_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArn,
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
			"customer_aws_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsNeptuneEventSubscriptionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn

	if v, ok := d.GetOk("name"); ok {
		d.Set("name", v.(string))
	} else if v, ok := d.GetOk("name_prefix"); ok {
		d.Set("name", resource.PrefixedUniqueId(v.(string)))
	} else {
		d.Set("name", resource.PrefixedUniqueId("tf-"))
	}

	tags := tagsFromMapNeptune(d.Get("tags").(map[string]interface{}))

	request := &neptune.CreateEventSubscriptionInput{
		SubscriptionName: aws.String(d.Get("name").(string)),
		SnsTopicArn:      aws.String(d.Get("sns_topic_arn").(string)),
		Enabled:          aws.Bool(d.Get("enabled").(bool)),
		Tags:             tags,
	}

	if v, ok := d.GetOk("source_ids"); ok {
		sourceIdsSet := v.(*schema.Set)
		sourceIds := make([]*string, sourceIdsSet.Len())
		for i, sourceId := range sourceIdsSet.List() {
			sourceIds[i] = aws.String(sourceId.(string))
		}
		request.SourceIds = sourceIds
	}

	if v, ok := d.GetOk("event_categories"); ok {
		eventCategoriesSet := v.(*schema.Set)
		eventCategories := make([]*string, eventCategoriesSet.Len())
		for i, eventCategory := range eventCategoriesSet.List() {
			eventCategories[i] = aws.String(eventCategory.(string))
		}
		request.EventCategories = eventCategories
	}

	if v, ok := d.GetOk("source_type"); ok {
		request.SourceType = aws.String(v.(string))
	}

	log.Println("[DEBUG] Create Neptune Event Subscription:", request)

	output, err := conn.CreateEventSubscription(request)
	if err != nil || output.EventSubscription == nil {
		return fmt.Errorf("Error creating Neptune Event Subscription %s: %s", d.Get("name").(string), err)
	}

	d.SetId(aws.StringValue(output.EventSubscription.CustSubscriptionId))

	log.Println("[INFO] Waiting for Neptune Event Subscription to be ready")

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating"},
		Target:     []string{"active"},
		Refresh:    resourceAwsNeptuneEventSubscriptionRefreshFunc(d.Id(), conn),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for Neptune Event Subscription state to be \"active\": %s", err)
	}

	return resourceAwsNeptuneEventSubscriptionRead(d, meta)
}

func resourceAwsNeptuneEventSubscriptionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn

	sub, err := resourceAwsNeptuneEventSubscriptionRetrieve(d.Id(), conn)
	if err != nil {
		return fmt.Errorf("Error reading Neptune Event Subscription %s: %s", d.Id(), err)
	}

	if sub == nil {
		log.Printf("[DEBUG] Neptune Event Subscription (%s) not found - removing from state", d.Id())
		d.SetId("")
		return nil
	}

	d.Set("arn", sub.EventSubscriptionArn)
	d.Set("name", sub.CustSubscriptionId)
	d.Set("sns_topic_arn", sub.SnsTopicArn)
	d.Set("enabled", sub.Enabled)
	d.Set("customer_aws_id", sub.CustomerAwsId)

	if sub.SourceType != nil {
		d.Set("source_type", sub.SourceType)
	}

	if sub.SourceIdsList != nil {
		if err := d.Set("source_ids", flattenStringList(sub.SourceIdsList)); err != nil {
			return fmt.Errorf("Error saving Source IDs to state for Neptune Event Subscription (%s): %s", d.Id(), err)
		}
	}

	if sub.EventCategoriesList != nil {
		if err := d.Set("event_categories", flattenStringList(sub.EventCategoriesList)); err != nil {
			return fmt.Errorf("Error saving Event Categories to state for Neptune Event Subscription (%s): %s", d.Id(), err)
		}
	}

	if err := saveTagsNeptune(conn, d, aws.StringValue(sub.EventSubscriptionArn)); err != nil {
		return fmt.Errorf("Error saving tags for Neptune Event Subscription (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceAwsNeptuneEventSubscriptionUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn

	d.Partial(true)
	requestUpdate := false

	req := &neptune.ModifyEventSubscriptionInput{
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

	if d.HasChange("sns_topic_arn") {
		req.SnsTopicArn = aws.String(d.Get("sns_topic_arn").(string))
		requestUpdate = true
	}

	if d.HasChange("source_type") {
		req.SourceType = aws.String(d.Get("source_type").(string))
		requestUpdate = true
	}

	log.Printf("[DEBUG] Send Neptune Event Subscription modification request: %#v", requestUpdate)
	if requestUpdate {
		log.Printf("[DEBUG] Neptune Event Subscription modification request: %#v", req)
		_, err := conn.ModifyEventSubscription(req)
		if err != nil {
			return fmt.Errorf("Modifying Neptune Event Subscription %s failed: %s", d.Id(), err)
		}

		log.Println("[INFO] Waiting for Neptune Event Subscription modification to finish")

		stateConf := &resource.StateChangeConf{
			Pending:    []string{"modifying"},
			Target:     []string{"active"},
			Refresh:    resourceAwsNeptuneEventSubscriptionRefreshFunc(d.Id(), conn),
			Timeout:    d.Timeout(schema.TimeoutUpdate),
			MinTimeout: 10 * time.Second,
			Delay:      30 * time.Second,
		}

		// Wait, catching any errors
		_, err = stateConf.WaitForState()
		if err != nil {
			return err
		}
		d.SetPartial("event_categories")
		d.SetPartial("enabled")
		d.SetPartial("sns_topic_arn")
		d.SetPartial("source_type")
	}

	if err := setTagsNeptune(conn, d, d.Get("arn").(string)); err != nil {
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
				_, err := conn.RemoveSourceIdentifierFromSubscription(&neptune.RemoveSourceIdentifierFromSubscriptionInput{
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
				_, err := conn.AddSourceIdentifierToSubscription(&neptune.AddSourceIdentifierToSubscriptionInput{
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

	return resourceAwsNeptuneEventSubscriptionRead(d, meta)
}

func resourceAwsNeptuneEventSubscriptionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).neptuneconn
	deleteOpts := neptune.DeleteEventSubscriptionInput{
		SubscriptionName: aws.String(d.Id()),
	}

	if _, err := conn.DeleteEventSubscription(&deleteOpts); err != nil {
		if isAWSErr(err, neptune.ErrCodeSubscriptionNotFoundFault, "") {
			log.Printf("[WARN] Neptune Event Subscription %s not found, removing from state", d.Id())
			d.SetId("")
			return nil
		}
		return fmt.Errorf("Error deleting Neptune Event Subscription %s: %s", d.Id(), err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"deleting"},
		Target:     []string{},
		Refresh:    resourceAwsNeptuneEventSubscriptionRefreshFunc(d.Id(), conn),
		Timeout:    d.Timeout(schema.TimeoutDelete),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	_, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error deleting Neptune Event Subscription %s: %s", d.Id(), err)
	}

	return nil
}

func resourceAwsNeptuneEventSubscriptionRefreshFunc(name string, conn *neptune.Neptune) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		sub, err := resourceAwsNeptuneEventSubscriptionRetrieve(name, conn)

		if err != nil {
			log.Printf("Error on retrieving Neptune Event Subscription when waiting: %s", err)
			return nil, "", err
		}

		if sub == nil {
			return nil, "", nil
		}

		if sub.Status != nil {
			log.Printf("[DEBUG] Neptune Event Subscription status for %s: %s", name, aws.StringValue(sub.Status))
		}

		return sub, aws.StringValue(sub.Status), nil
	}
}

func resourceAwsNeptuneEventSubscriptionRetrieve(name string, conn *neptune.Neptune) (*neptune.EventSubscription, error) {

	request := &neptune.DescribeEventSubscriptionsInput{
		SubscriptionName: aws.String(name),
	}

	describeResp, err := conn.DescribeEventSubscriptions(request)
	if err != nil {
		if isAWSErr(err, neptune.ErrCodeSubscriptionNotFoundFault, "") {
			log.Printf("[DEBUG] Neptune Event Subscription (%s) not found", name)
			return nil, nil
		}
		return nil, err
	}

	if len(describeResp.EventSubscriptionsList) != 1 ||
		aws.StringValue(describeResp.EventSubscriptionsList[0].CustSubscriptionId) != name {
		return nil, nil
	}

	return describeResp.EventSubscriptionsList[0], nil
}
