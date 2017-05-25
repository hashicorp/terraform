package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsElastiCacheCluster() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsElastiCacheClusterRead,

		Schema: map[string]*schema.Schema{
			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(v interface{}) string {
					value := v.(string)
					return strings.ToLower(value)
				},
			},

			"node_type": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"num_cache_nodes": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"subnet_group_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"engine": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"engine_version": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"parameter_group_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"replication_group_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"security_group_names": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"security_group_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"maintenance_window": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"snapshot_window": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"snapshot_retention_limit": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"availability_zone": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"notification_topic_arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"configuration_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cluster_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cache_nodes": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"port": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"availability_zone": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},

			"tags": tagsSchemaComputed(),
		},
	}
}

func dataSourceAwsElastiCacheClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	req := &elasticache.DescribeCacheClustersInput{
		CacheClusterId:    aws.String(d.Get("cluster_id").(string)),
		ShowCacheNodeInfo: aws.Bool(true),
	}

	resp, err := conn.DescribeCacheClusters(req)
	if err != nil {
		return err
	}

	if len(resp.CacheClusters) < 1 {
		return fmt.Errorf("Your query returned no results. Please change your search criteria and try again.")
	}
	if len(resp.CacheClusters) > 1 {
		return fmt.Errorf("Your query returned more than one result. Please try a more specific search criteria.")
	}

	cluster := resp.CacheClusters[0]

	d.SetId(*cluster.CacheClusterId)

	d.Set("cluster_id", cluster.CacheClusterId)
	d.Set("node_type", cluster.CacheNodeType)
	d.Set("num_cache_nodes", cluster.NumCacheNodes)
	d.Set("subnet_group_name", cluster.CacheSubnetGroupName)
	d.Set("engine", cluster.Engine)
	d.Set("engine_version", cluster.EngineVersion)
	d.Set("security_group_names", flattenElastiCacheSecurityGroupNames(cluster.CacheSecurityGroups))
	d.Set("security_group_ids", flattenElastiCacheSecurityGroupIds(cluster.SecurityGroups))

	if cluster.CacheParameterGroup != nil {
		d.Set("parameter_group_name", cluster.CacheParameterGroup.CacheParameterGroupName)
	}

	if cluster.ReplicationGroupId != nil {
		d.Set("replication_group_id", cluster.ReplicationGroupId)
	}

	d.Set("maintenance_window", cluster.PreferredMaintenanceWindow)
	d.Set("snapshot_window", cluster.SnapshotWindow)
	d.Set("snapshot_retention_limit", cluster.SnapshotRetentionLimit)
	d.Set("availability_zone", cluster.PreferredAvailabilityZone)

	if cluster.NotificationConfiguration != nil {
		if *cluster.NotificationConfiguration.TopicStatus == "active" {
			d.Set("notification_topic_arn", cluster.NotificationConfiguration.TopicArn)
		}
	}

	if cluster.ConfigurationEndpoint != nil {
		d.Set("port", cluster.ConfigurationEndpoint.Port)
		d.Set("configuration_endpoint", aws.String(fmt.Sprintf("%s:%d", *cluster.ConfigurationEndpoint.Address, *cluster.ConfigurationEndpoint.Port)))
		d.Set("cluster_address", aws.String(fmt.Sprintf("%s", *cluster.ConfigurationEndpoint.Address)))
	}

	if err := setCacheNodeData(d, cluster); err != nil {
		return err
	}

	arn, err := buildECARN(d.Id(), meta.(*AWSClient).partition, meta.(*AWSClient).accountid, meta.(*AWSClient).region)
	if err != nil {
		log.Printf("[DEBUG] Error building ARN for ElastiCache Cluster %s", *cluster.CacheClusterId)
	}
	d.Set("arn", arn)

	tagResp, err := conn.ListTagsForResource(&elasticache.ListTagsForResourceInput{
		ResourceName: aws.String(arn),
	})

	if err != nil {
		log.Printf("[DEBUG] Error retrieving tags for ARN: %s", arn)
	}

	var et []*elasticache.Tag
	if len(tagResp.TagList) > 0 {
		et = tagResp.TagList
	}
	d.Set("tags", tagsToMapEC(et))

	return nil

}
