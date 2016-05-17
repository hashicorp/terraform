package aws

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsIotTopicRule() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsIotTopicRuleCreate,
		Read:   resourceAwsIotTopicRuleRead,
		Update: resourceAwsIotTopicRuleUpdate,
		Delete: resourceAwsIotTopicRuleDelete,
		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"enabled": &schema.Schema{
				Type: schema.TypeBool,
			},
			"sql": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"cloudwatch_alarm": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"alarm_name": &schema.Schema{
							Type: schema.TypeString,
						},
						"role_arn": &schema.Schema{
							Type: schema.TypeString,
						},
						"state_reason": &schema.Schema{
							Type: schema.TypeString,
						},
						"state_value": &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
			"cloudwatch_metric": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"metric_name": &schema.Schema{
							Type: schema.TypeString,
						},
						"metric_namespace": &schema.Schema{
							Type: schema.TypeString,
						},
						"metric_timestamp": &schema.Schema{
							Type: schema.TypeString,
						},
						"metric_unit": &schema.Schema{
							Type: schema.TypeString,
						},
						"metric_value": &schema.Schema{
							Type: schema.TypeString,
						},
						"role_arn": &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
			"dynamodb": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hash_key_field": &schema.Schema{
							Type: schema.TypeString,
						},
						"hash_key_value": &schema.Schema{
							Type: schema.TypeString,
						},
						"payload_field": &schema.Schema{
							Type: schema.TypeString,
						},
						"range_key_field": &schema.Schema{
							Type: schema.TypeString,
						},
						"range_key_value": &schema.Schema{
							Type: schema.TypeString,
						},
						"role_arn": &schema.Schema{
							Type: schema.TypeString,
						},
						"table_name": &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
			"elasticsearch": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"endpoint": &schema.Schema{
							Type: schema.TypeString,
						},
						"id": &schema.Schema{
							Type: schema.TypeString,
						},
						"index": &schema.Schema{
							Type: schema.TypeString,
						},
						"role_arn": &schema.Schema{
							Type: schema.TypeString,
						},
						"type": &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
			"firehose": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"delivery_stream_name": &schema.Schema{
							Type: schema.TypeString,
						},
						"role_arn": &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
			"kinesis": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"partition_key": &schema.Schema{
							Type: schema.TypeString,
						},
						"role_arn": &schema.Schema{
							Type: schema.TypeString,
						},
						"stream_name": &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
			"lambda": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"function_arn": &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
			"republish": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"role_arn": &schema.Schema{
							Type: schema.TypeString,
						},
						"topic": &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
			"s3": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bucket_name": &schema.Schema{
							Type: schema.TypeString,
						},
						"key": &schema.Schema{
							Type: schema.TypeString,
						},
						"role_arn": &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
			"sns": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"message_format": &schema.Schema{
							Type: schema.TypeString,
						},
						"target_arn": &schema.Schema{
							Type: schema.TypeString,
						},
						"role_arn": &schema.Schema{
							Type: schema.TypeString,
						},
					},
				},
			},
			"sqs": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"queue_url": &schema.Schema{
							Type: schema.TypeString,
						},
						"role_arn": &schema.Schema{
							Type: schema.TypeString,
						},
						"use_base64": &schema.Schema{
							Type: schema.TypeBool,
						},
					},
				},
			},
		},
	}
}

func resourceAwsIotTopicRuleCreate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsIotTopicRuleRead(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsIotTopicRuleUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceAwsIotTopicRuleDelete(d *schema.ResourceData, meta interface{}) error {
	return nil
}
