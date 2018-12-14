package aws

import (
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/mq"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsMqBroker() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsmQBrokerRead,

		Schema: map[string]*schema.Schema{
			"broker_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"broker_name"},
			},
			"broker_name": {
				Type:          schema.TypeString,
				Optional:      true,
				Computed:      true,
				ConflictsWith: []string{"broker_id"},
			},
			"auto_minor_version_upgrade": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"configuration": {
				Type:     schema.TypeList,
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"revision": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"deployment_mode": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"engine_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"engine_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"host_instance_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"instances": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"console_url": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ip_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"endpoints": {
							Type:     schema.TypeList,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
					},
				},
			},
			"logs": {
				Type:     schema.TypeList,
				Optional: true,
				MaxItems: 1,
				// Ignore missing configuration block
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if old == "1" && new == "0" {
						return true
					}
					return false
				},
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"general": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"audit": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
			"maintenance_window_start_time": {
				Type:     schema.TypeList,
				MaxItems: 1,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"day_of_week": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"time_of_day": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"time_zone": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"publicly_accessible": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"security_groups": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"subnet_ids": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
			},
			"user": {
				Type:     schema.TypeSet,
				Computed: true,
				Set:      resourceAwsMqUserHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"console_access": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"groups": {
							Type:     schema.TypeSet,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
							Computed: true,
						},
						"username": {
							Type:     schema.TypeString,
							Required: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceAwsmQBrokerRead(d *schema.ResourceData, meta interface{}) error {
	if brokerId, ok := d.GetOk("broker_id"); ok {
		d.SetId(brokerId.(string))
	} else {
		conn := meta.(*AWSClient).mqconn
		brokerName := d.Get("broker_name").(string)
		var nextToken string
		for {
			out, err := conn.ListBrokers(&mq.ListBrokersInput{NextToken: aws.String(nextToken)})
			if err != nil {
				return errors.New("Failed to list mq brokers")
			}
			for _, broker := range out.BrokerSummaries {
				if aws.StringValue(broker.BrokerName) == brokerName {
					brokerId := aws.StringValue(broker.BrokerId)
					d.Set("broker_id", brokerId)
					d.SetId(brokerId)
				}
			}
			if out.NextToken == nil {
				break
			}
			nextToken = *out.NextToken
		}

		if d.Id() == "" {
			return fmt.Errorf("Failed to determine mq broker: %s", brokerName)
		}
	}

	return resourceAwsMqBrokerRead(d, meta)
}
