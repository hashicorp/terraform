package aws

import (
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsElasticacheCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticacheClusterCreate,
		Read:   resourceAwsElasticacheClusterRead,
		Update: resourceAwsElasticacheClusterUpdate,
		Delete: resourceAwsElasticacheClusterDelete,

		Schema: map[string]*schema.Schema{
			"cluster_id": &schema.Schema{
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
			"configuration_endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"engine": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"node_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"num_cache_nodes": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
			},
			"parameter_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"engine_version": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"maintenance_window": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(val interface{}) string {
					// Elasticache always changes the maintenance
					// to lowercase
					return strings.ToLower(val.(string))
				},
			},
			"subnet_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"security_group_names": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
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
			// Exported Attributes
			"cache_nodes": &schema.Schema{
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
					},
				},
			},
			"notification_topic_arn": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			// A single-element string list containing an Amazon Resource Name (ARN) that
			// uniquely identifies a Redis RDB snapshot file stored in Amazon S3. The snapshot
			// file will be used to populate the node group.
			//
			// See also:
			// https://github.com/aws/aws-sdk-go/blob/4862a174f7fc92fb523fc39e68f00b87d91d2c3d/service/elasticache/api.go#L2079
			"snapshot_arns": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"snapshot_window": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"snapshot_retention_limit": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(int)
					if value > 35 {
						es = append(es, fmt.Errorf(
							"snapshot retention limit cannot be more than 35 days"))
					}
					return
				},
			},

			"az_mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"availability_zone": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"availability_zones": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"tags": tagsSchema(),

			// apply_immediately is used to determine when the update modifications
			// take place.
			// See http://docs.aws.amazon.com/AmazonElastiCache/latest/APIReference/API_ModifyCacheCluster.html
			"apply_immediately": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsElasticacheClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	clusterId := d.Get("cluster_id").(string)
	nodeType := d.Get("node_type").(string)           // e.g) cache.m1.small
	numNodes := int64(d.Get("num_cache_nodes").(int)) // 2
	engine := d.Get("engine").(string)                // memcached
	engineVersion := d.Get("engine_version").(string) // 1.4.14
	port := int64(d.Get("port").(int))                // e.g) 11211
	subnetGroupName := d.Get("subnet_group_name").(string)
	securityNameSet := d.Get("security_group_names").(*schema.Set)
	securityIdSet := d.Get("security_group_ids").(*schema.Set)

	securityNames := expandStringList(securityNameSet.List())
	securityIds := expandStringList(securityIdSet.List())

	tags := tagsFromMapEC(d.Get("tags").(map[string]interface{}))
	req := &elasticache.CreateCacheClusterInput{
		CacheClusterId:          aws.String(clusterId),
		CacheNodeType:           aws.String(nodeType),
		NumCacheNodes:           aws.Int64(numNodes),
		Engine:                  aws.String(engine),
		EngineVersion:           aws.String(engineVersion),
		Port:                    aws.Int64(port),
		CacheSubnetGroupName:    aws.String(subnetGroupName),
		CacheSecurityGroupNames: securityNames,
		SecurityGroupIds:        securityIds,
		Tags:                    tags,
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

	if v, ok := d.GetOk("az_mode"); ok {
		req.AZMode = aws.String(v.(string))
	}

	if v, ok := d.GetOk("availability_zone"); ok {
		req.PreferredAvailabilityZone = aws.String(v.(string))
	}

	preferred_azs := d.Get("availability_zones").(*schema.Set).List()
	if len(preferred_azs) > 0 {
		azs := expandStringList(preferred_azs)
		req.PreferredAvailabilityZones = azs
	}

	resp, err := conn.CreateCacheCluster(req)
	if err != nil {
		return fmt.Errorf("Error creating Elasticache: %s", err)
	}

	// Assign the cluster id as the resource ID
	// Elasticache always retains the id in lower case, so we have to
	// mimic that or else we won't be able to refresh a resource whose
	// name contained uppercase characters.
	d.SetId(strings.ToLower(*resp.CacheCluster.CacheClusterId))

	pending := []string{"creating"}
	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     []string{"available"},
		Refresh:    cacheClusterStateRefreshFunc(conn, d.Id(), "available", pending),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	log.Printf("[DEBUG] Waiting for state to become available: %v", d.Id())
	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for elasticache (%s) to be created: %s", d.Id(), sterr)
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
		if eccErr, ok := err.(awserr.Error); ok && eccErr.Code() == "CacheClusterNotFound" {
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
		}

		d.Set("subnet_group_name", c.CacheSubnetGroupName)
		d.Set("security_group_names", c.CacheSecurityGroups)
		d.Set("security_group_ids", c.SecurityGroups)
		d.Set("parameter_group_name", c.CacheParameterGroup)
		d.Set("maintenance_window", c.PreferredMaintenanceWindow)
		d.Set("snapshot_window", c.SnapshotWindow)
		d.Set("snapshot_retention_limit", c.SnapshotRetentionLimit)
		if c.NotificationConfiguration != nil {
			if *c.NotificationConfiguration.TopicStatus == "active" {
				d.Set("notification_topic_arn", c.NotificationConfiguration.TopicArn)
			}
		}
		d.Set("availability_zone", c.PreferredAvailabilityZone)

		if err := setCacheNodeData(d, c); err != nil {
			return err
		}
		// list tags for resource
		// set tags
		arn, err := buildECARN(d, meta)
		if err != nil {
			log.Printf("[DEBUG] Error building ARN for ElastiCache Cluster, not setting Tags for cluster %s", *c.CacheClusterId)
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
	}

	return nil
}

func resourceAwsElasticacheClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn
	arn, err := buildECARN(d, meta)
	if err != nil {
		log.Printf("[DEBUG] Error building ARN for ElastiCache Cluster, not updating Tags for cluster %s", d.Id())
	} else {
		if err := setTagsEC(conn, d, arn); err != nil {
			return err
		}
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

	if d.HasChange("snapshot_retention_limit") {
		req.SnapshotRetentionLimit = aws.Int64(int64(d.Get("snapshot_retention_limit").(int)))
		requestUpdate = true
	}

	if d.HasChange("num_cache_nodes") {
		oraw, nraw := d.GetChange("num_cache_nodes")
		o := oraw.(int)
		n := nraw.(int)
		if v, ok := d.GetOk("az_mode"); ok && v.(string) == "cross-az" && n == 1 {
			return fmt.Errorf("[WARN] Error updateing Elasticache cluster (%s), error: Cross-AZ mode is not supported in a single cache node.", d.Id())
		}
		if n < o {
			log.Printf("[INFO] Cluster %s is marked for Decreasing cache nodes from %d to %d", d.Id(), o, n)
			nodesToRemove := getCacheNodesToRemove(d, o, o-n)
			req.CacheNodeIdsToRemove = nodesToRemove
		}

		req.NumCacheNodes = aws.Int64(int64(d.Get("num_cache_nodes").(int)))
		requestUpdate = true

	}

	if requestUpdate {
		log.Printf("[DEBUG] Modifying ElastiCache Cluster (%s), opts:\n%s", d.Id(), req)
		_, err := conn.ModifyCacheCluster(req)
		if err != nil {
			return fmt.Errorf("[WARN] Error updating ElastiCache cluster (%s), error: %s", d.Id(), err)
		}

		log.Printf("[DEBUG] Waiting for update: %s", d.Id())
		pending := []string{"modifying", "rebooting cache cluster nodes", "snapshotting"}
		stateConf := &resource.StateChangeConf{
			Pending:    pending,
			Target:     []string{"available"},
			Refresh:    cacheClusterStateRefreshFunc(conn, d.Id(), "available", pending),
			Timeout:    5 * time.Minute,
			Delay:      5 * time.Second,
			MinTimeout: 3 * time.Second,
		}

		_, sterr := stateConf.WaitForState()
		if sterr != nil {
			return fmt.Errorf("Error waiting for elasticache (%s) to update: %s", d.Id(), sterr)
		}
	}

	return resourceAwsElasticacheClusterRead(d, meta)
}

func getCacheNodesToRemove(d *schema.ResourceData, oldNumberOfNodes int, cacheNodesToRemove int) []*string {
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

	req := &elasticache.DeleteCacheClusterInput{
		CacheClusterId: aws.String(d.Id()),
	}
	_, err := conn.DeleteCacheCluster(req)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Waiting for deletion: %v", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "available", "deleting", "incompatible-parameters", "incompatible-network", "restore-failed"},
		Target:     []string{},
		Refresh:    cacheClusterStateRefreshFunc(conn, d.Id(), "", []string{}),
		Timeout:    20 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for elasticache (%s) to delete: %s", d.Id(), sterr)
	}

	d.SetId("")

	return nil
}

func cacheClusterStateRefreshFunc(conn *elasticache.ElastiCache, clusterID, givenState string, pending []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{
			CacheClusterId:    aws.String(clusterID),
			ShowCacheNodeInfo: aws.Bool(true),
		})
		if err != nil {
			apierr := err.(awserr.Error)
			log.Printf("[DEBUG] message: %v, code: %v", apierr.Message(), apierr.Code())
			if apierr.Message() == fmt.Sprintf("CacheCluster not found: %v", clusterID) {
				log.Printf("[DEBUG] Detect deletion")
				return nil, "", nil
			}

			log.Printf("[ERROR] CacheClusterStateRefreshFunc: %s", err)
			return nil, "", err
		}

		if len(resp.CacheClusters) == 0 {
			return nil, "", fmt.Errorf("[WARN] Error: no Cache Clusters found for id (%s)", clusterID)
		}

		var c *elasticache.CacheCluster
		for _, cluster := range resp.CacheClusters {
			if *cluster.CacheClusterId == clusterID {
				log.Printf("[DEBUG] Found matching ElastiCache cluster: %s", *cluster.CacheClusterId)
				c = cluster
			}
		}

		if c == nil {
			return nil, "", fmt.Errorf("[WARN] Error: no matching Elastic Cache cluster for id (%s)", clusterID)
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

func buildECARN(d *schema.ResourceData, meta interface{}) (string, error) {
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
