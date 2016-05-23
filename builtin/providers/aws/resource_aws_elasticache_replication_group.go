package aws

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsElasticacheReplicationGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticacheReplicationGroupCreate,
		Read:   resourceAwsElasticacheReplicationGroupRead,
		Update: resourceAwsElasticacheReplicationGroupUpdate,
		Delete: resourceAwsElasticacheReplicationGroupDelete,

		Schema: map[string]*schema.Schema{
			"replication_group_id": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateElastiCacheClusterId,
			},
			"replication_group_description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"primary_cluster_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"automatic_failover_enabled": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"number_cache_clusters": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
			},
			"cache_clusters": &schema.Schema{
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"address": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"port": &schema.Schema{
							Type:     schema.TypeInt,
							Computed: true,
						},
						"availability_zone": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
						"role": &schema.Schema{
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"primary_endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"preferred_cache_cluster_azs": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"cache_node_type": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"engine": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "redis",
			},
			"engine_version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"cache_parameter_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"cache_subnet_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"cache_security_group_names": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"security_group_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"snapshot_arns": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"snapshot_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"preferred_maintenance_window": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"notification_topic_arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"notification_topic_status": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"auto_minor_version_upgrade": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"snapshot_retention_limit": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  0,
			},
			"snapshot_window": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"apply_immediately": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"snapshotting_cluster_id": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"tags": tagsSchema(),
		},
	}
}

func resourceAwsElasticacheReplicationGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	params := &elasticache.CreateReplicationGroupInput{
		ReplicationGroupId:          aws.String(d.Get("replication_group_id").(string)),
		ReplicationGroupDescription: aws.String(d.Get("replication_group_description").(string)),
		Engine: aws.String(d.Get("engine").(string)),
	}

	if v, ok := d.GetOk("primary_cluster_id"); ok {
		params.PrimaryClusterId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("automatic_failover_enabled"); ok {
		params.AutomaticFailoverEnabled = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("number_cache_clusters"); ok {
		params.NumCacheClusters = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("preferred_cache_cluster_azs"); ok {
		params.PreferredCacheClusterAZs = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("cache_node_type"); ok {
		params.CacheNodeType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("engine_version"); ok {
		params.EngineVersion = aws.String(v.(string))
	}

	if v, ok := d.GetOk("cache_parameter_group_name"); ok {
		params.CacheParameterGroupName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("cache_subnet_group_name"); ok {
		params.CacheSubnetGroupName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("cache_security_group_names"); ok {
		params.CacheSecurityGroupNames = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("security_group_ids"); ok {
		params.SecurityGroupIds = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("snapshot_arns"); ok {
		params.SnapshotArns = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := d.GetOk("snapshot_name"); ok {
		params.SnapshotName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("preferred_maintenance_window"); ok {
		params.PreferredMaintenanceWindow = aws.String(v.(string))
	}

	if v, ok := d.GetOk("port"); ok {
		params.Port = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("notification_topic_arn"); ok {
		params.NotificationTopicArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("auto_minor_version_upgrade"); ok {
		params.AutoMinorVersionUpgrade = aws.Bool(v.(bool))
	}

	if v, ok := d.GetOk("snapshot_retention_limit"); ok {
		params.SnapshotRetentionLimit = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("snapshot_window"); ok {
		params.SnapshotWindow = aws.String(v.(string))
	}

	resp, err := conn.CreateReplicationGroup(params)
	if err != nil {
		return fmt.Errorf("Error creating Elasticache Replication Group: %s", err)
	}

	d.SetId(*resp.ReplicationGroup.ReplicationGroupId)

	pending := []string{"creating", "modifying"}
	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     []string{"available"},
		Refresh:    cacheClusterReplicationGroupStateRefreshFunc(conn, d.Id(), "available", pending),
		Timeout:    30 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	log.Printf("[DEBUG] Waiting for state to become available: %v", d.Id())
	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for elasticache replication group (%s) to be created: %s", d.Id(), sterr)
	}

	return resourceAwsElasticacheReplicationGroupRead(d, meta)
}

func resourceAwsElasticacheReplicationGroupRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn
	req := &elasticache.DescribeReplicationGroupsInput{
		ReplicationGroupId: aws.String(d.Id()),
	}

	res, err := conn.DescribeReplicationGroups(req)
	if err != nil {
		if eccErr, ok := err.(awserr.Error); ok && eccErr.Code() == "ReplicationGroupNotFound" {
			log.Printf("[WARN] Elasticache Replication Group (%s) not found", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	if len(res.ReplicationGroups) == 1 {
		rg := res.ReplicationGroups[0]
		fmt.Printf("%v\n", rg)

		d.Set("replication_group_id", rg.ReplicationGroupId)

		if *rg.AutomaticFailover == "enabled" {
			d.Set("automatic_failover_enabled", true)
		} else {
			d.Set("automatic_failover_enabled", false)
		}

		d.Set("number_cache_clusters", len(rg.MemberClusters))

		if len(rg.NodeGroups) == 1 {
			groupMembers := rg.NodeGroups[0].NodeGroupMembers
			cacheNodeData := make([]map[string]interface{}, 0, len(groupMembers))

			for _, node := range groupMembers {
				if node.CacheClusterId == nil || node.ReadEndpoint == nil || node.ReadEndpoint.Address == nil || node.ReadEndpoint.Port == nil || node.PreferredAvailabilityZone == nil || node.CurrentRole == nil {
					return fmt.Errorf("Unexpected nil pointer in: %s", node)
				}
				cacheNodeData = append(cacheNodeData, map[string]interface{}{
					"id":                *node.CacheClusterId,
					"address":           *node.ReadEndpoint.Address,
					"port":              int(*node.ReadEndpoint.Port),
					"availability_zone": *node.PreferredAvailabilityZone,
					"role":              *node.CurrentRole,
				})
			}
			d.Set("cache_clusters", cacheNodeData)

			primaryEndpoint := rg.NodeGroups[0].PrimaryEndpoint
			if primaryEndpoint.Address == nil || primaryEndpoint.Port == nil {
				return fmt.Errorf("Unexpected nil pointer in: %s", primaryEndpoint)
			}
			d.Set("primary_endpoint", fmt.Sprintf("%s:%d", *primaryEndpoint.Address, *primaryEndpoint.Port))
		}

		/*
			            // TODO: copied from cache_cluster but returns empty tags
						// list tags for resource set tags
						arn, err := buildECARN(d, meta)
						if err != nil {
							log.Printf("[DEBUG] Error building ARN for ElastiCache Cluster, not setting Tags for ReplicationGroup %s", *rg.ReplicationGroupId)
						} else {
							resp, err := conn.ListTagsForResource(&elasticache.ListTagsForResourceInput{
								ResourceName: aws.String(arn),
							})

							if err != nil {
								log.Printf("[DEBUG] Error retrieving tags for ARN: %s", arn)
							}

							var et []*elasticache.Tag
							if len(resp.TagList) > 0 {
								et = resp.TagList
							}
							d.Set("tags", tagsToMapEC(et))
						}
		*/
	}

	return nil
}

func resourceAwsElasticacheReplicationGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	params := &elasticache.ModifyReplicationGroupInput{
		ApplyImmediately:   aws.Bool(d.Get("apply_immediately").(bool)),
		ReplicationGroupId: aws.String(d.Id()),
	}

	if d.HasChange("replication_group_description") {
		params.ReplicationGroupDescription = aws.String(d.Get("description").(string))
	}

	if d.HasChange("primary_cluster_id") {
		params.PrimaryClusterId = aws.String(d.Get("primary_cluster_id").(string))
	}

	if d.HasChange("snapshotting_cluster_id") {
		params.SnapshottingClusterId = aws.String(d.Get("snapshotting_cluster_id").(string))
	}

	if d.HasChange("automatic_failover_enabled") {
		params.AutomaticFailoverEnabled = aws.Bool(d.Get("automatic_failover").(bool))
	}

	if d.HasChange("cache_security_group_names") {
		params.CacheSecurityGroupNames = expandStringList(d.Get("cache_security_group_names").(*schema.Set).List())
	}

	if d.HasChange("security_group_ids") {
		params.SecurityGroupIds = expandStringList(d.Get("security_group_ids").(*schema.Set).List())
	}

	if d.HasChange("preferred_maintenance_window") {
		params.PreferredMaintenanceWindow = aws.String(d.Get("preferred_maintenance_window").(string))
	}

	if d.HasChange("notification_topic_arn") {
		params.NotificationTopicArn = aws.String(d.Get("notification_topic_arn").(string))
	}

	if d.HasChange("cache_parameter_group_name") {
		params.CacheParameterGroupName = aws.String(d.Get("cache_parameter_group_name").(string))
	}

	if d.HasChange("notification_topic_status") {
		params.NotificationTopicStatus = aws.String(d.Get("notification_topic_status").(string))
	}

	if d.HasChange("engine_version") {
		params.EngineVersion = aws.String(d.Get("engine_version").(string))
	}

	if d.HasChange("auto_minor_version_upgrade") {
		params.AutoMinorVersionUpgrade = aws.Bool(d.Get("auto_minor_version_upgrade").(bool))
	}

	if d.HasChange("snapshot_retention_limit") {
		params.SnapshotRetentionLimit = aws.Int64(int64(d.Get("snapshot_retention_limit").(int)))
	}

	if d.HasChange("snapshot_window") {
		params.SnapshotWindow = aws.String(d.Get("snapshot_window").(string))
	}

	_, err := conn.ModifyReplicationGroup(params)
	if err != nil {
		return fmt.Errorf("Error updating Elasticache replication group: %s", err)
	}

	return resourceAwsElasticacheReplicationGroupRead(d, meta)
}

func resourceAwsElasticacheReplicationGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	req := &elasticache.DeleteReplicationGroupInput{
		ReplicationGroupId: aws.String(d.Id()),
	}

	_, err := conn.DeleteReplicationGroup(req)
	if err != nil {
		if ec2err, ok := err.(awserr.Error); ok && ec2err.Code() == "ReplicationGroupNotFoundFault" {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error deleting Elasticache replication group: %s", err)
	}

	log.Printf("[DEBUG] Waiting for deletion: %v", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "modifying", "available", "deleting"},
		Target:     []string{},
		Refresh:    cacheClusterReplicationGroupStateRefreshFunc(conn, d.Id(), "", []string{}),
		Timeout:    15 * time.Minute,
		Delay:      20 * time.Second,
		MinTimeout: 5 * time.Second,
	}

	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for replication group (%s) to delete: %s", d.Id(), sterr)
	}

	return nil
}

func cacheClusterReplicationGroupStateRefreshFunc(conn *elasticache.ElastiCache, replicationGroupId, givenState string, pending []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{
			ReplicationGroupId: aws.String(replicationGroupId),
		})
		if err != nil {
			apierr := err.(awserr.Error)
			log.Printf("[DEBUG] message: %v, code: %v", apierr.Message(), apierr.Code())
			if apierr.Message() == fmt.Sprintf("ReplicationGroup %v not found.", replicationGroupId) {
				log.Printf("[DEBUG] Detect deletion")
				return nil, "", nil
			}

			log.Printf("[ERROR] cacheClusterReplicationGroupStateRefreshFunc: %s", err)
			return nil, "", err
		}

		if len(resp.ReplicationGroups) == 0 {
			return nil, "", fmt.Errorf("[WARN] Error: no Cache Replication Groups found for id (%s)", replicationGroupId)
		}

		var rg *elasticache.ReplicationGroup
		for _, replicationGroup := range resp.ReplicationGroups {
			if *replicationGroup.ReplicationGroupId == replicationGroupId {
				log.Printf("[DEBUG] Found matching ElastiCache Replication Group: %s", *replicationGroup.ReplicationGroupId)
				rg = replicationGroup
			}
		}

		if rg == nil {
			return nil, "", fmt.Errorf("[WARN] Error: no matching Elasticcache Replication Group for id (%s)", replicationGroupId)
		}

		log.Printf("[DEBUG] ElastiCache Replication Group (%s) status: %v", replicationGroupId, *rg.Status)

		// return the current state if it's in the pending array
		for _, p := range pending {
			log.Printf("[DEBUG] ElastiCache: checking pending state (%s) for Replication Group (%s), Replication Group status: %s", pending, replicationGroupId, *rg.Status)
			s := *rg.Status
			if p == s {
				log.Printf("[DEBUG] Return with status: %v", *rg.Status)
				return s, p, nil
			}
		}

		//		// return given state if it's not in pending
		//		if givenState != "" {
		//			log.Printf("[DEBUG] ElastiCache: checking given state (%s) of Replication Group (%s) against Replication Group status (%s)", givenState, replicationGroupId, *rg.Status)
		//			// check to make sure we have the node count we're expecting
		//			if int64(len(rg.)) != *c.NumCacheNodes {
		//				log.Printf("[DEBUG] Node count is not what is expected: %d found, %d expected", len(c.CacheNodes), *c.NumCacheNodes)
		//				return nil, "creating", nil
		//			}
		//
		//			log.Printf("[DEBUG] Node count matched (%d)", len(c.CacheNodes))
		//			// loop the nodes and check their status as well
		//			for _, n := range c.CacheNodes {
		//				log.Printf("[DEBUG] Checking cache node for status: %s", n)
		//				if n.CacheNodeStatus != nil && *n.CacheNodeStatus != "available" {
		//					log.Printf("[DEBUG] Node (%s) is not yet available, status: %s", *n.CacheNodeId, *n.CacheNodeStatus)
		//					return nil, "creating", nil
		//				}
		//				log.Printf("[DEBUG] Cache node not in expected state")
		//			}
		//			log.Printf("[DEBUG] ElastiCache returning given state (%s), cluster: %s", givenState, c)
		//			return c, givenState, nil
		//		}
		//		log.Printf("[DEBUG] current status: %v", *c.CacheClusterStatus)
		return rg, *rg.Status, nil
	}
}

func buildECReplicationGroupARN(d *schema.ResourceData, meta interface{}) (string, error) {
	iamconn := meta.(*AWSClient).iamconn
	region := meta.(*AWSClient).region
	// An zero value GetUserInput{} defers to the currently logged in user
	resp, err := iamconn.GetUser(&iam.GetUserInput{})
	if err != nil {
		return "", err
	}
	userARN := *resp.User.Arn
	accountID := strings.Split(userARN, ":")[4]
	arn := fmt.Sprintf("arn:aws:elasticache:%s:%s:cluster:%s", region, accountID, d.Id())
	return arn, nil
}
