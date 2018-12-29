package aws

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/service/elasticache"
	gversion "github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform/helper/customdiff"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsElasticacheCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticacheClusterCreate,
		Read:   resourceAwsElasticacheClusterRead,
		Update: resourceAwsElasticacheClusterUpdate,
		Delete: resourceAwsElasticacheClusterDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"apply_immediately": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"availability_zone": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"availability_zones": {
				Type:          schema.TypeSet,
				Optional:      true,
				ForceNew:      true,
				Elem:          &schema.Schema{Type: schema.TypeString},
				Set:           schema.HashString,
				ConflictsWith: []string{"preferred_availability_zones"},
				Deprecated:    "Use `preferred_availability_zones` instead",
			},
			"az_mode": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ValidateFunc: validation.StringInSlice([]string{
					elasticache.AZModeCrossAz,
					elasticache.AZModeSingleAz,
				}, false),
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
			"cluster_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cluster_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				StateFunc: func(val interface{}) string {
					// Elasticache normalizes cluster ids to lowercase,
					// so we have to do this too or else we can end up
					// with non-converging diffs.
					return strings.ToLower(val.(string))
				},
				ValidateFunc: validateElastiCacheClusterId,
			},
			"configuration_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"engine": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"engine_version": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"maintenance_window": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(val interface{}) string {
					// Elasticache always changes the maintenance
					// to lowercase
					return strings.ToLower(val.(string))
				},
				ValidateFunc: validateOnceAWeekWindowFormat,
			},
			"node_type": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"notification_topic_arn": {
				Type:     schema.TypeString,
				Optional: true,
			},
			"num_cache_nodes": {
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},
			"parameter_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"port": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Suppress default memcached/redis ports when not defined
					if !d.IsNewResource() && new == "0" && (old == "6379" || old == "11211") {
						return true
					}
					return false
				},
			},
			"preferred_availability_zones": {
				Type:     schema.TypeList,
				Optional: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"replication_group_id": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				ConflictsWith: []string{
					"availability_zones",
					"az_mode",
					"engine_version",
					"engine",
					"maintenance_window",
					"node_type",
					"notification_topic_arn",
					"num_cache_nodes",
					"parameter_group_name",
					"port",
					"security_group_ids",
					"security_group_names",
					"snapshot_arns",
					"snapshot_name",
					"snapshot_retention_limit",
					"snapshot_window",
					"subnet_group_name",
				},
				Computed: true,
			},
			"security_group_names": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"security_group_ids": {
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			// A single-element string list containing an Amazon Resource Name (ARN) that
			// uniquely identifies a Redis RDB snapshot file stored in Amazon S3. The snapshot
			// file will be used to populate the node group.
			//
			// See also:
			// https://github.com/aws/aws-sdk-go/blob/4862a174f7fc92fb523fc39e68f00b87d91d2c3d/service/elasticache/api.go#L2079
			"snapshot_arns": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"snapshot_retention_limit": {
				Type:         schema.TypeInt,
				Optional:     true,
				ValidateFunc: validation.IntAtMost(35),
			},
			"snapshot_window": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				ValidateFunc: validateOnceADayWindowFormat,
			},
			"snapshot_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"subnet_group_name": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"tags": tagsSchema(),
		},

		CustomizeDiff: customdiff.Sequence(
			func(diff *schema.ResourceDiff, v interface{}) error {
				// Plan time validation for az_mode
				// InvalidParameterCombination: Must specify at least two cache nodes in order to specify AZ Mode of 'cross-az'.
				if v, ok := diff.GetOk("az_mode"); !ok || v.(string) != elasticache.AZModeCrossAz {
					return nil
				}
				if v, ok := diff.GetOk("num_cache_nodes"); !ok || v.(int) != 1 {
					return nil
				}
				return errors.New(`az_mode "cross-az" is not supported with num_cache_nodes = 1`)
			},
			func(diff *schema.ResourceDiff, v interface{}) error {
				// Plan time validation for engine_version
				// InvalidParameterCombination: Cannot modify memcached from 1.4.33 to 1.4.24
				// InvalidParameterCombination: Cannot modify redis from 3.2.6 to 3.2.4
				if diff.Id() == "" || !diff.HasChange("engine_version") {
					return nil
				}
				o, n := diff.GetChange("engine_version")
				oVersion, err := gversion.NewVersion(o.(string))
				if err != nil {
					return err
				}
				nVersion, err := gversion.NewVersion(n.(string))
				if err != nil {
					return err
				}
				if nVersion.GreaterThan(oVersion) {
					return nil
				}
				return diff.ForceNew("engine_version")
			},
			func(diff *schema.ResourceDiff, v interface{}) error {
				// Plan time validation for num_cache_nodes
				// InvalidParameterValue: Cannot create a Redis cluster with a NumCacheNodes parameter greater than 1.
				if v, ok := diff.GetOk("engine"); !ok || v.(string) == "memcached" {
					return nil
				}
				if v, ok := diff.GetOk("num_cache_nodes"); !ok || v.(int) == 1 {
					return nil
				}
				return errors.New(`engine "redis" does not support num_cache_nodes > 1`)
			},
			func(diff *schema.ResourceDiff, v interface{}) error {
				// Engine memcached does not currently support vertical scaling
				// InvalidParameterCombination: Scaling is not supported for engine memcached
				// https://docs.aws.amazon.com/AmazonElastiCache/latest/UserGuide/Scaling.Memcached.html#Scaling.Memcached.Vertically
				if diff.Id() == "" || !diff.HasChange("node_type") {
					return nil
				}
				if v, ok := diff.GetOk("engine"); !ok || v.(string) == "redis" {
					return nil
				}
				return diff.ForceNew("node_type")
			},
		),
	}
}

func resourceAwsElasticacheClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	req := &elasticache.CreateCacheClusterInput{}

	if v, ok := d.GetOk("replication_group_id"); ok {
		req.ReplicationGroupId = aws.String(v.(string))
	} else {
		securityNameSet := d.Get("security_group_names").(*schema.Set)
		securityIdSet := d.Get("security_group_ids").(*schema.Set)
		securityNames := expandStringList(securityNameSet.List())
		securityIds := expandStringList(securityIdSet.List())
		tags := tagsFromMapEC(d.Get("tags").(map[string]interface{}))

		req.CacheSecurityGroupNames = securityNames
		req.SecurityGroupIds = securityIds
		req.Tags = tags
	}

	if v, ok := d.GetOk("cluster_id"); ok {
		req.CacheClusterId = aws.String(v.(string))
	}

	if v, ok := d.GetOk("node_type"); ok {
		req.CacheNodeType = aws.String(v.(string))
	}

	if v, ok := d.GetOk("num_cache_nodes"); ok {
		req.NumCacheNodes = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("engine"); ok {
		req.Engine = aws.String(v.(string))
	}

	if v, ok := d.GetOk("engine_version"); ok {
		req.EngineVersion = aws.String(v.(string))
	}

	if v, ok := d.GetOk("port"); ok {
		req.Port = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("subnet_group_name"); ok {
		req.CacheSubnetGroupName = aws.String(v.(string))
	}

	// parameter groups are optional and can be defaulted by AWS
	if v, ok := d.GetOk("parameter_group_name"); ok {
		req.CacheParameterGroupName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("snapshot_retention_limit"); ok {
		req.SnapshotRetentionLimit = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("snapshot_window"); ok {
		req.SnapshotWindow = aws.String(v.(string))
	}

	if v, ok := d.GetOk("maintenance_window"); ok {
		req.PreferredMaintenanceWindow = aws.String(v.(string))
	}

	if v, ok := d.GetOk("notification_topic_arn"); ok {
		req.NotificationTopicArn = aws.String(v.(string))
	}

	snaps := d.Get("snapshot_arns").(*schema.Set).List()
	if len(snaps) > 0 {
		s := expandStringList(snaps)
		req.SnapshotArns = s
		log.Printf("[DEBUG] Restoring Redis cluster from S3 snapshot: %#v", s)
	}

	if v, ok := d.GetOk("snapshot_name"); ok {
		req.SnapshotName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("az_mode"); ok {
		req.AZMode = aws.String(v.(string))
	}

	if v, ok := d.GetOk("availability_zone"); ok {
		req.PreferredAvailabilityZone = aws.String(v.(string))
	}

	if v, ok := d.GetOk("preferred_availability_zones"); ok && len(v.([]interface{})) > 0 {
		req.PreferredAvailabilityZones = expandStringList(v.([]interface{}))
	} else {
		preferred_azs := d.Get("availability_zones").(*schema.Set).List()
		if len(preferred_azs) > 0 {
			azs := expandStringList(preferred_azs)
			req.PreferredAvailabilityZones = azs
		}
	}

	id, err := createElasticacheCacheCluster(conn, req)
	if err != nil {
		return fmt.Errorf("error creating Elasticache Cache Cluster: %s", err)
	}

	d.SetId(id)

	err = waitForCreateElasticacheCacheCluster(conn, d.Id(), 40*time.Minute)
	if err != nil {
		return fmt.Errorf("error waiting for Elasticache Cache Cluster (%s) to be created: %s", d.Id(), err)
	}

	return resourceAwsElasticacheClusterRead(d, meta)
}

func resourceAwsElasticacheClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn
	req := &elasticache.DescribeCacheClustersInput{
		CacheClusterId:    aws.String(d.Id()),
		ShowCacheNodeInfo: aws.Bool(true),
	}

	res, err := conn.DescribeCacheClusters(req)
	if err != nil {
		if isAWSErr(err, elasticache.ErrCodeCacheClusterNotFoundFault, "") {
			log.Printf("[WARN] ElastiCache Cluster (%s) not found", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	if len(res.CacheClusters) == 1 {
		c := res.CacheClusters[0]
		d.Set("cluster_id", c.CacheClusterId)
		d.Set("node_type", c.CacheNodeType)
		d.Set("num_cache_nodes", c.NumCacheNodes)
		d.Set("engine", c.Engine)
		d.Set("engine_version", c.EngineVersion)
		if c.ConfigurationEndpoint != nil {
			d.Set("port", c.ConfigurationEndpoint.Port)
			d.Set("configuration_endpoint", aws.String(fmt.Sprintf("%s:%d", *c.ConfigurationEndpoint.Address, *c.ConfigurationEndpoint.Port)))
			d.Set("cluster_address", aws.String(fmt.Sprintf("%s", *c.ConfigurationEndpoint.Address)))
		} else if len(c.CacheNodes) > 0 {
			d.Set("port", int(aws.Int64Value(c.CacheNodes[0].Endpoint.Port)))
		}

		if c.ReplicationGroupId != nil {
			d.Set("replication_group_id", c.ReplicationGroupId)
		}

		d.Set("subnet_group_name", c.CacheSubnetGroupName)
		d.Set("security_group_names", flattenElastiCacheSecurityGroupNames(c.CacheSecurityGroups))
		d.Set("security_group_ids", flattenElastiCacheSecurityGroupIds(c.SecurityGroups))
		if c.CacheParameterGroup != nil {
			d.Set("parameter_group_name", c.CacheParameterGroup.CacheParameterGroupName)
		}
		d.Set("maintenance_window", c.PreferredMaintenanceWindow)
		d.Set("snapshot_window", c.SnapshotWindow)
		d.Set("snapshot_retention_limit", c.SnapshotRetentionLimit)
		if c.NotificationConfiguration != nil {
			if *c.NotificationConfiguration.TopicStatus == "active" {
				d.Set("notification_topic_arn", c.NotificationConfiguration.TopicArn)
			}
		}
		d.Set("availability_zone", c.PreferredAvailabilityZone)
		if *c.PreferredAvailabilityZone == "Multiple" {
			d.Set("az_mode", "cross-az")
		} else {
			d.Set("az_mode", "single-az")
		}

		if err := setCacheNodeData(d, c); err != nil {
			return err
		}
		// list tags for resource
		// set tags
		arn := arn.ARN{
			Partition: meta.(*AWSClient).partition,
			Service:   "elasticache",
			Region:    meta.(*AWSClient).region,
			AccountID: meta.(*AWSClient).accountid,
			Resource:  fmt.Sprintf("cluster:%s", d.Id()),
		}.String()
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

	return nil
}

func resourceAwsElasticacheClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	arn := arn.ARN{
		Partition: meta.(*AWSClient).partition,
		Service:   "elasticache",
		Region:    meta.(*AWSClient).region,
		AccountID: meta.(*AWSClient).accountid,
		Resource:  fmt.Sprintf("cluster:%s", d.Id()),
	}.String()
	if err := setTagsEC(conn, d, arn); err != nil {
		return err
	}

	req := &elasticache.ModifyCacheClusterInput{
		CacheClusterId:   aws.String(d.Id()),
		ApplyImmediately: aws.Bool(d.Get("apply_immediately").(bool)),
	}

	requestUpdate := false
	if d.HasChange("security_group_ids") {
		if attr := d.Get("security_group_ids").(*schema.Set); attr.Len() > 0 {
			req.SecurityGroupIds = expandStringList(attr.List())
			requestUpdate = true
		}
	}

	if d.HasChange("parameter_group_name") {
		req.CacheParameterGroupName = aws.String(d.Get("parameter_group_name").(string))
		requestUpdate = true
	}

	if d.HasChange("maintenance_window") {
		req.PreferredMaintenanceWindow = aws.String(d.Get("maintenance_window").(string))
		requestUpdate = true
	}

	if d.HasChange("notification_topic_arn") {
		v := d.Get("notification_topic_arn").(string)
		req.NotificationTopicArn = aws.String(v)
		if v == "" {
			inactive := "inactive"
			req.NotificationTopicStatus = &inactive
		}
		requestUpdate = true
	}

	if d.HasChange("engine_version") {
		req.EngineVersion = aws.String(d.Get("engine_version").(string))
		requestUpdate = true
	}

	if d.HasChange("snapshot_window") {
		req.SnapshotWindow = aws.String(d.Get("snapshot_window").(string))
		requestUpdate = true
	}

	if d.HasChange("node_type") {
		req.CacheNodeType = aws.String(d.Get("node_type").(string))
		requestUpdate = true
	}

	if d.HasChange("snapshot_retention_limit") {
		req.SnapshotRetentionLimit = aws.Int64(int64(d.Get("snapshot_retention_limit").(int)))
		requestUpdate = true
	}

	if d.HasChange("az_mode") {
		req.AZMode = aws.String(d.Get("az_mode").(string))
		requestUpdate = true
	}

	if d.HasChange("num_cache_nodes") {
		oraw, nraw := d.GetChange("num_cache_nodes")
		o := oraw.(int)
		n := nraw.(int)
		if n < o {
			log.Printf("[INFO] Cluster %s is marked for Decreasing cache nodes from %d to %d", d.Id(), o, n)
			nodesToRemove := getCacheNodesToRemove(o, o-n)
			req.CacheNodeIdsToRemove = nodesToRemove
		} else {
			log.Printf("[INFO] Cluster %s is marked for increasing cache nodes from %d to %d", d.Id(), o, n)
			// SDK documentation for NewAvailabilityZones states:
			// The list of Availability Zones where the new Memcached cache nodes are created.
			//
			// This parameter is only valid when NumCacheNodes in the request is greater
			// than the sum of the number of active cache nodes and the number of cache
			// nodes pending creation (which may be zero). The number of Availability Zones
			// supplied in this list must match the cache nodes being added in this request.
			if v, ok := d.GetOk("preferred_availability_zones"); ok && len(v.([]interface{})) > 0 {
				// Here we check the list length to prevent a potential panic :)
				if len(v.([]interface{})) != n {
					return fmt.Errorf("length of preferred_availability_zones (%d) must match num_cache_nodes (%d)", len(v.([]interface{})), n)
				}
				req.NewAvailabilityZones = expandStringList(v.([]interface{})[o:])
			}
		}

		req.NumCacheNodes = aws.Int64(int64(d.Get("num_cache_nodes").(int)))
		requestUpdate = true

	}

	if requestUpdate {
		log.Printf("[DEBUG] Modifying ElastiCache Cluster (%s), opts:\n%s", d.Id(), req)
		_, err := conn.ModifyCacheCluster(req)
		if err != nil {
			return fmt.Errorf("Error updating ElastiCache cluster (%s), error: %s", d.Id(), err)
		}

		log.Printf("[DEBUG] Waiting for update: %s", d.Id())
		pending := []string{"modifying", "rebooting cache cluster nodes", "snapshotting"}
		stateConf := &resource.StateChangeConf{
			Pending:    pending,
			Target:     []string{"available"},
			Refresh:    cacheClusterStateRefreshFunc(conn, d.Id(), "available", pending),
			Timeout:    80 * time.Minute,
			MinTimeout: 10 * time.Second,
			Delay:      30 * time.Second,
		}

		_, sterr := stateConf.WaitForState()
		if sterr != nil {
			return fmt.Errorf("Error waiting for elasticache (%s) to update: %s", d.Id(), sterr)
		}
	}

	return resourceAwsElasticacheClusterRead(d, meta)
}

func getCacheNodesToRemove(oldNumberOfNodes int, cacheNodesToRemove int) []*string {
	nodesIdsToRemove := []*string{}
	for i := oldNumberOfNodes; i > oldNumberOfNodes-cacheNodesToRemove && i > 0; i-- {
		s := fmt.Sprintf("%04d", i)
		nodesIdsToRemove = append(nodesIdsToRemove, &s)
	}

	return nodesIdsToRemove
}

func setCacheNodeData(d *schema.ResourceData, c *elasticache.CacheCluster) error {
	sortedCacheNodes := make([]*elasticache.CacheNode, len(c.CacheNodes))
	copy(sortedCacheNodes, c.CacheNodes)
	sort.Sort(byCacheNodeId(sortedCacheNodes))

	cacheNodeData := make([]map[string]interface{}, 0, len(sortedCacheNodes))

	for _, node := range sortedCacheNodes {
		if node.CacheNodeId == nil || node.Endpoint == nil || node.Endpoint.Address == nil || node.Endpoint.Port == nil || node.CustomerAvailabilityZone == nil {
			return fmt.Errorf("Unexpected nil pointer in: %s", node)
		}
		cacheNodeData = append(cacheNodeData, map[string]interface{}{
			"id":                *node.CacheNodeId,
			"address":           *node.Endpoint.Address,
			"port":              int(*node.Endpoint.Port),
			"availability_zone": *node.CustomerAvailabilityZone,
		})
	}

	return d.Set("cache_nodes", cacheNodeData)
}

type byCacheNodeId []*elasticache.CacheNode

func (b byCacheNodeId) Len() int      { return len(b) }
func (b byCacheNodeId) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byCacheNodeId) Less(i, j int) bool {
	return b[i].CacheNodeId != nil && b[j].CacheNodeId != nil &&
		*b[i].CacheNodeId < *b[j].CacheNodeId
}

func resourceAwsElasticacheClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	err := deleteElasticacheCacheCluster(conn, d.Id())
	if err != nil {
		if isAWSErr(err, elasticache.ErrCodeCacheClusterNotFoundFault, "") {
			return nil
		}
		return fmt.Errorf("error deleting Elasticache Cache Cluster (%s): %s", d.Id(), err)
	}
	err = waitForDeleteElasticacheCacheCluster(conn, d.Id(), 40*time.Minute)
	if err != nil {
		return fmt.Errorf("error waiting for Elasticache Cache Cluster (%s) to be deleted: %s", d.Id(), err)
	}

	return nil
}

func cacheClusterStateRefreshFunc(conn *elasticache.ElastiCache, clusterID, givenState string, pending []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{
			CacheClusterId:    aws.String(clusterID),
			ShowCacheNodeInfo: aws.Bool(true),
		})
		if err != nil {
			if isAWSErr(err, elasticache.ErrCodeCacheClusterNotFoundFault, "") {
				log.Printf("[DEBUG] Detect deletion")
				return nil, "", nil
			}

			log.Printf("[ERROR] CacheClusterStateRefreshFunc: %s", err)
			return nil, "", err
		}

		if len(resp.CacheClusters) == 0 {
			return nil, "", fmt.Errorf("Error: no Cache Clusters found for id (%s)", clusterID)
		}

		var c *elasticache.CacheCluster
		for _, cluster := range resp.CacheClusters {
			if *cluster.CacheClusterId == clusterID {
				log.Printf("[DEBUG] Found matching ElastiCache cluster: %s", *cluster.CacheClusterId)
				c = cluster
			}
		}

		if c == nil {
			return nil, "", fmt.Errorf("Error: no matching Elastic Cache cluster for id (%s)", clusterID)
		}

		log.Printf("[DEBUG] ElastiCache Cluster (%s) status: %v", clusterID, *c.CacheClusterStatus)

		// return the current state if it's in the pending array
		for _, p := range pending {
			log.Printf("[DEBUG] ElastiCache: checking pending state (%s) for cluster (%s), cluster status: %s", pending, clusterID, *c.CacheClusterStatus)
			s := *c.CacheClusterStatus
			if p == s {
				log.Printf("[DEBUG] Return with status: %v", *c.CacheClusterStatus)
				return c, p, nil
			}
		}

		// return given state if it's not in pending
		if givenState != "" {
			log.Printf("[DEBUG] ElastiCache: checking given state (%s) of cluster (%s) against cluster status (%s)", givenState, clusterID, *c.CacheClusterStatus)
			// check to make sure we have the node count we're expecting
			if int64(len(c.CacheNodes)) != *c.NumCacheNodes {
				log.Printf("[DEBUG] Node count is not what is expected: %d found, %d expected", len(c.CacheNodes), *c.NumCacheNodes)
				return nil, "creating", nil
			}

			log.Printf("[DEBUG] Node count matched (%d)", len(c.CacheNodes))
			// loop the nodes and check their status as well
			for _, n := range c.CacheNodes {
				log.Printf("[DEBUG] Checking cache node for status: %s", n)
				if n.CacheNodeStatus != nil && *n.CacheNodeStatus != "available" {
					log.Printf("[DEBUG] Node (%s) is not yet available, status: %s", *n.CacheNodeId, *n.CacheNodeStatus)
					return nil, "creating", nil
				}
				log.Printf("[DEBUG] Cache node not in expected state")
			}
			log.Printf("[DEBUG] ElastiCache returning given state (%s), cluster: %s", givenState, c)
			return c, givenState, nil
		}
		log.Printf("[DEBUG] current status: %v", *c.CacheClusterStatus)
		return c, *c.CacheClusterStatus, nil
	}
}

func createElasticacheCacheCluster(conn *elasticache.ElastiCache, input *elasticache.CreateCacheClusterInput) (string, error) {
	log.Printf("[DEBUG] Creating Elasticache Cache Cluster: %s", input)
	output, err := conn.CreateCacheCluster(input)
	if err != nil {
		return "", err
	}
	if output == nil || output.CacheCluster == nil {
		return "", errors.New("missing cluster ID after creation")
	}
	// Elasticache always retains the id in lower case, so we have to
	// mimic that or else we won't be able to refresh a resource whose
	// name contained uppercase characters.
	return strings.ToLower(aws.StringValue(output.CacheCluster.CacheClusterId)), nil
}

func waitForCreateElasticacheCacheCluster(conn *elasticache.ElastiCache, cacheClusterID string, timeout time.Duration) error {
	pending := []string{"creating", "modifying", "restoring", "snapshotting"}
	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     []string{"available"},
		Refresh:    cacheClusterStateRefreshFunc(conn, cacheClusterID, "available", pending),
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	log.Printf("[DEBUG] Waiting for Elasticache Cache Cluster (%s) to be created", cacheClusterID)
	_, err := stateConf.WaitForState()
	return err
}

func deleteElasticacheCacheCluster(conn *elasticache.ElastiCache, cacheClusterID string) error {
	input := &elasticache.DeleteCacheClusterInput{
		CacheClusterId: aws.String(cacheClusterID),
	}
	log.Printf("[DEBUG] Deleting Elasticache Cache Cluster: %s", input)
	err := resource.Retry(5*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteCacheCluster(input)
		if err != nil {
			// This will not be fixed by retrying
			if isAWSErr(err, elasticache.ErrCodeInvalidCacheClusterStateFault, "serving as primary") {
				return resource.NonRetryableError(err)
			}
			// The cluster may be just snapshotting, so we retry until it's ready for deletion
			if isAWSErr(err, elasticache.ErrCodeInvalidCacheClusterStateFault, "") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	return err
}

func waitForDeleteElasticacheCacheCluster(conn *elasticache.ElastiCache, cacheClusterID string, timeout time.Duration) error {
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "available", "deleting", "incompatible-parameters", "incompatible-network", "restore-failed", "snapshotting"},
		Target:     []string{},
		Refresh:    cacheClusterStateRefreshFunc(conn, cacheClusterID, "", []string{}),
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}
	log.Printf("[DEBUG] Waiting for Elasticache Cache Cluster deletion: %v", cacheClusterID)

	_, err := stateConf.WaitForState()
	return err
}
