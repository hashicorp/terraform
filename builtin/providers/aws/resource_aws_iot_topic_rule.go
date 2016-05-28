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
				Type:     schema.TypeBool,
				Optional: true,
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
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"state_reason": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"state_value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
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
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_namespace": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_timestamp": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_unit": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"metric_value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
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
							Type:     schema.TypeString,
							Required: true,
						},
						"hash_key_value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"payload_field": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"range_key_field": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"range_key_value": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"table_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
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
							Type:     schema.TypeString,
							Required: true,
						},
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"index": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"type": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
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
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
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
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"stream_name": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
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
							Type:     schema.TypeString,
							Required: true,
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
							Type:     schema.TypeString,
							Required: true,
						},
						"topic": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
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
							Type:     schema.TypeString,
							Required: true,
						},
						"key": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
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
							Type:     schema.TypeString,
							Required: true,
						},
						"target_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
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
							Type:     schema.TypeString,
							Required: true,
						},
						"role_arn": &schema.Schema{
							Type:     schema.TypeString,
							Required: true,
						},
						"use_base64": &schema.Schema{
							Type:     schema.TypeBool,
							Required: true,
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
