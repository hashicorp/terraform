package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsElasticacheReplicationGroup() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsElasticacheReplicationGroupRead,
		Schema: map[string]*schema.Schema{
			"replication_group_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"replication_group_description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"auth_token_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"automatic_failover_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"configuration_endpoint_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"primary_endpoint_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"number_cache_clusters": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"member_clusters": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"node_type": {
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
		},
	}
}

func dataSourceAwsElasticacheReplicationGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn
	input := &elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: aws.String(d.Get("replication_group_id").(string)),
	}

	log.Printf("[DEBUG] Reading ElastiCache Replication Group: %s", input)
	resp, err := conn.DescribeReplicationGroups(input)
	if err != nil {
		return err
	}

	var rg *elasticache.ReplicationGroup
	for _, r := range resp.ReplicationGroups {
		if *r.ReplicationGroupId == d.Get("replication_group_id").(string) {
			rg = r
		}
	}
	if rg == nil {
		return fmt.Errorf("Elasticache Replication Group (%s) not found", d.Get("replication_group_id").(string))
	}

	d.SetId(*rg.ReplicationGroupId)
	d.Set("replication_group_description", rg.Description)
	d.Set("auth_token_enabled", rg.AuthTokenEnabled)
	if rg.AutomaticFailover != nil {
		switch *rg.AutomaticFailover {
		case elasticache.AutomaticFailoverStatusDisabled, elasticache.AutomaticFailoverStatusDisabling:
			d.Set("automatic_failover_enabled", false)
		case elasticache.AutomaticFailoverStatusEnabled, elasticache.AutomaticFailoverStatusEnabling:
			d.Set("automatic_failover_enabled", true)
		}
	}
	if rg.ConfigurationEndpoint != nil {
		d.Set("port", rg.ConfigurationEndpoint.Port)
		d.Set("configuration_endpoint_address", rg.ConfigurationEndpoint.Address)
	} else {
		if rg.NodeGroups == nil {
			d.SetId("")
			return fmt.Errorf("Elasticache Replication Group (%s) doesn't have node groups.", d.Get("replication_group_id").(string))
		}
		d.Set("port", rg.NodeGroups[0].PrimaryEndpoint.Port)
		d.Set("primary_endpoint_address", rg.NodeGroups[0].PrimaryEndpoint.Address)
	}
	d.Set("number_cache_clusters", len(rg.MemberClusters))
	if err := d.Set("member_clusters", flattenStringList(rg.MemberClusters)); err != nil {
		return fmt.Errorf("error setting member_clusters: %s", err)
	}
	d.Set("node_type", rg.CacheNodeType)
	d.Set("snapshot_window", rg.SnapshotWindow)
	d.Set("snapshot_retention_limit", rg.SnapshotRetentionLimit)
	return nil
}
