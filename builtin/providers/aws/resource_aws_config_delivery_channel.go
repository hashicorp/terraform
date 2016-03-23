package aws

import (
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/configservice"
)

func resourceAwsConfigDeliveryChannel() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsConfigDeliveryChannelPut,
		Read:   resourceAwsConfigDeliveryChannelRead,
		Update: resourceAwsConfigDeliveryChannelPut,
		Delete: resourceAwsConfigDeliveryChannelDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "default",
			},
			"s3_bucket_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"s3_key_prefix": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"sns_topic_arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"config_snapshot_delivery_properties": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delivery_frequency": &schema.Schema{
							Type:     schema.TypeString,
							Optional: true,
						},
					},
				},
			},
		},
	}
}

func resourceAwsConfigDeliveryChannelPut(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	name := d.Get("name").(string)
	channel := configservice.DeliveryChannel{
		Name: aws.String(name),
	}

	if v, ok := d.GetOk("s3_bucket_name"); ok {
		channel.S3BucketName = aws.String(v.(string))
	}
	if v, ok := d.GetOk("s3_key_prefix"); ok {
		channel.S3KeyPrefix = aws.String(v.(string))
	}
	if v, ok := d.GetOk("sns_topic_arn"); ok {
		channel.SnsTopicARN = aws.String(v.(string))
	}

	if p, ok := d.GetOk("config_snapshot_delivery_properties"); ok {
		propertiesBlocks := p.([]interface{})

		properties := configservice.ConfigSnapshotDeliveryProperties{}
		block := propertiesBlocks[0].(map[string]interface{})

		if v, ok := block["delivery_frequency"]; ok {
			properties.DeliveryFrequency = aws.String(v.(string))
		}

		channel.ConfigSnapshotDeliveryProperties = &properties
	}

	input := configservice.PutDeliveryChannelInput{DeliveryChannel: &channel}
	_, err := conn.PutDeliveryChannel(&input)
	if err != nil {
		return fmt.Errorf("Creating Delivery Channel failed: %s", err)
	}

	d.SetId(name)

	return resourceAwsConfigDeliveryChannelRead(d, meta)
}

func resourceAwsConfigDeliveryChannelRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn

	input := configservice.DescribeDeliveryChannelsInput{
		DeliveryChannelNames: []*string{aws.String(d.Id())},
	}
	out, err := conn.DescribeDeliveryChannels(&input)
	if err != nil {
		return fmt.Errorf("Getting Delivery Channel failed: %s", err)
	}

	if len(out.DeliveryChannels) < 1 {
		log.Printf("[WARN] Delivery Channel %q is gone", d.Id())
		d.SetId("")
		return nil
	}

	if len(out.DeliveryChannels) > 1 {
		return fmt.Errorf("Received more than 1 configuration recorders (expected exactly 1): %s",
			out.DeliveryChannels)
	}

	channel := out.DeliveryChannels[0]

	d.Set("name", channel.Name)
	d.Set("s3_bucket_name", channel.S3BucketName)
	d.Set("s3_key_prefix", channel.S3KeyPrefix)
	d.Set("sns_topic_arn", channel.SnsTopicARN)

	if channel.ConfigSnapshotDeliveryProperties != nil {
		d.Set("config_snapshot_delivery_properties", flattenConfigSnapshotDeliveryProperties(channel.ConfigSnapshotDeliveryProperties))
	}

	return nil
}

func resourceAwsConfigDeliveryChannelDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).configconn
	input := configservice.DeleteDeliveryChannelInput{
		DeliveryChannelName: aws.String(d.Id()),
	}
	_, err := conn.DeleteDeliveryChannel(&input)
	if err != nil {
		return fmt.Errorf("Stopping Delivery Channel failed: %s", err)
	}

	d.SetId("")
	return nil
}
