package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsKinesisStream() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsKinesisStreamRead,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"arn": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"creation_timestamp": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"status": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"retention_period": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"open_shards": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"closed_shards": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"shard_level_metrics": &schema.Schema{
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"tags": &schema.Schema{
				Type:     schema.TypeMap,
				Computed: true,
			},
		},
	}
}

func dataSourceAwsKinesisStreamRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).kinesisconn
	sn := d.Get("name").(string)

	state, err := readKinesisStreamState(conn, sn)
	if err != nil {
		return err
	}
	d.SetId(state.arn)
	d.Set("arn", state.arn)
	d.Set("name", sn)
	d.Set("open_shards", state.openShards)
	d.Set("closed_shards", state.closedShards)
	d.Set("status", state.status)
	d.Set("creation_timestamp", state.creationTimestamp)
	d.Set("retention_period", state.retentionPeriod)
	d.Set("shard_level_metrics", state.shardLevelMetrics)

	tags, err := conn.ListTagsForStream(&kinesis.ListTagsForStreamInput{
		StreamName: aws.String(sn),
	})
	if err != nil {
		return err
	}
	d.Set("tags", tagsToMapKinesis(tags.Tags))

	return nil
}
