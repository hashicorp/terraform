package aws

import (
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsSpotDataFeedSubscription() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsSpotDataFeedSubscriptionCreate,
		Read:   resourceAwsSpotDataFeedSubscriptionRead,
		Delete: resourceAwsSpotDataFeedSubscriptionDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"bucket": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"prefix": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsSpotDataFeedSubscriptionCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	params := &ec2.CreateSpotDatafeedSubscriptionInput{
		Bucket: aws.String(d.Get("bucket").(string)),
	}

	if v, ok := d.GetOk("prefix"); ok {
		params.Prefix = aws.String(v.(string))
	}

	log.Printf("[INFO] Creating Spot Datafeed Subscription")
	_, err := conn.CreateSpotDatafeedSubscription(params)
	if err != nil {
		return errwrap.Wrapf("Error Creating Spot Datafeed Subscription: {{err}}", err)
	}

	d.SetId("spot-datafeed-subscription")

	return resourceAwsSpotDataFeedSubscriptionRead(d, meta)
}
func resourceAwsSpotDataFeedSubscriptionRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	resp, err := conn.DescribeSpotDatafeedSubscription(&ec2.DescribeSpotDatafeedSubscriptionInput{})
	if err != nil {
		cgw, ok := err.(awserr.Error)
		if ok && cgw.Code() == "InvalidSpotDatafeed.NotFound" {
			log.Printf("[WARNING] Spot Datafeed Subscription Not Found so refreshing from state")
			d.SetId("")
			return nil
		}
		return errwrap.Wrapf("Error Describing Spot Datafeed Subscription: {{err}}", err)
	}

	if resp == nil {
		log.Printf("[WARNING] Spot Datafeed Subscription Not Found so refreshing from state")
		d.SetId("")
		return nil
	}

	subscription := *resp.SpotDatafeedSubscription
	d.Set("bucket", subscription.Bucket)
	d.Set("prefix", subscription.Prefix)

	return nil
}
func resourceAwsSpotDataFeedSubscriptionDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).ec2conn

	log.Printf("[INFO] Deleting Spot Datafeed Subscription")
	_, err := conn.DeleteSpotDatafeedSubscription(&ec2.DeleteSpotDatafeedSubscriptionInput{})
	if err != nil {
		return errwrap.Wrapf("Error deleting Spot Datafeed Subscription: {{err}}", err)
	}
	return nil
}
