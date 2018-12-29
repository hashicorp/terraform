package aws

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elasticache"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsElasticacheReplicationGroup() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsElasticacheReplicationGroupCreate,
		Read:   resourceAwsElasticacheReplicationGroupRead,
		Update: resourceAwsElasticacheReplicationGroupUpdate,
		Delete: resourceAwsElasticacheReplicationGroupDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"apply_immediately": {
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},
			"at_rest_encryption_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
			"auth_token": {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ForceNew:     true,
				ValidateFunc: validateAwsElastiCacheReplicationGroupAuthToken,
			},
			"auto_minor_version_upgrade": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"automatic_failover_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"availability_zones": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
			"cluster_mode": {
				Type:     schema.TypeList,
				Optional: true,
				// We allow Computed: true here since using number_cache_clusters
				// and a cluster mode enabled parameter_group_name will create
				// a single shard replication group with number_cache_clusters - 1
				// read replicas. Otherwise, the resource is marked ForceNew.
				Computed: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"replicas_per_node_group": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},
						"num_node_groups": {
							Type:     schema.TypeInt,
							Required: true,
						},
					},
				},
			},
			"configuration_endpoint_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"engine": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Default:      "redis",
				ValidateFunc: validateAwsElastiCacheReplicationGroupEngine,
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
			"member_clusters": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
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
			"number_cache_clusters": {
				Type:     schema.TypeInt,
				Computed: true,
				Optional: true,
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
			"primary_endpoint_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"replication_group_description": {
				Type:     schema.TypeString,
				Required: true,
			},
			"replication_group_id": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateAwsElastiCacheReplicationGroupId,
				StateFunc: func(val interface{}) string {
					return strings.ToLower(val.(string))
				},
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
			"transit_encryption_enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},
		},
		SchemaVersion: 1,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(60 * time.Minute),
			Delete: schema.DefaultTimeout(40 * time.Minute),
			Update: schema.DefaultTimeout(40 * time.Minute),
		},
	}
}

func resourceAwsElasticacheReplicationGroupCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	tags := tagsFromMapEC(d.Get("tags").(map[string]interface{}))
	params := &elasticache.CreateReplicationGroupInput{
		ReplicationGroupId:          aws.String(d.Get("replication_group_id").(string)),
		ReplicationGroupDescription: aws.String(d.Get("replication_group_description").(string)),
		AutomaticFailoverEnabled:    aws.Bool(d.Get("automatic_failover_enabled").(bool)),
		AutoMinorVersionUpgrade:     aws.Bool(d.Get("auto_minor_version_upgrade").(bool)),
		CacheNodeType:               aws.String(d.Get("node_type").(string)),
		Engine:                      aws.String(d.Get("engine").(string)),
		Tags:                        tags,
	}

	if v, ok := d.GetOk("engine_version"); ok {
		params.EngineVersion = aws.String(v.(string))
	}

	preferred_azs := d.Get("availability_zones").(*schema.Set).List()
	if len(preferred_azs) > 0 {
		azs := expandStringList(preferred_azs)
		params.PreferredCacheClusterAZs = azs
	}

	if v, ok := d.GetOk("parameter_group_name"); ok {
		params.CacheParameterGroupName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("port"); ok {
		params.Port = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("subnet_group_name"); ok {
		params.CacheSubnetGroupName = aws.String(v.(string))
	}

	security_group_names := d.Get("security_group_names").(*schema.Set).List()
	if len(security_group_names) > 0 {
		params.CacheSecurityGroupNames = expandStringList(security_group_names)
	}

	security_group_ids := d.Get("security_group_ids").(*schema.Set).List()
	if len(security_group_ids) > 0 {
		params.SecurityGroupIds = expandStringList(security_group_ids)
	}

	snaps := d.Get("snapshot_arns").(*schema.Set).List()
	if len(snaps) > 0 {
		params.SnapshotArns = expandStringList(snaps)
	}

	if v, ok := d.GetOk("maintenance_window"); ok {
		params.PreferredMaintenanceWindow = aws.String(v.(string))
	}

	if v, ok := d.GetOk("notification_topic_arn"); ok {
		params.NotificationTopicArn = aws.String(v.(string))
	}

	if v, ok := d.GetOk("snapshot_retention_limit"); ok {
		params.SnapshotRetentionLimit = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("snapshot_window"); ok {
		params.SnapshotWindow = aws.String(v.(string))
	}

	if v, ok := d.GetOk("snapshot_name"); ok {
		params.SnapshotName = aws.String(v.(string))
	}

	if _, ok := d.GetOk("transit_encryption_enabled"); ok {
		params.TransitEncryptionEnabled = aws.Bool(d.Get("transit_encryption_enabled").(bool))
	}

	if _, ok := d.GetOk("at_rest_encryption_enabled"); ok {
		params.AtRestEncryptionEnabled = aws.Bool(d.Get("at_rest_encryption_enabled").(bool))
	}

	if v, ok := d.GetOk("auth_token"); ok {
		params.AuthToken = aws.String(v.(string))
	}

	clusterMode, clusterModeOk := d.GetOk("cluster_mode")
	cacheClusters, cacheClustersOk := d.GetOk("number_cache_clusters")

	if !clusterModeOk && !cacheClustersOk || clusterModeOk && cacheClustersOk {
		return fmt.Errorf("Either `number_cache_clusters` or `cluster_mode` must be set")
	}

	if clusterModeOk {
		clusterModeList := clusterMode.([]interface{})
		attributes := clusterModeList[0].(map[string]interface{})

		if v, ok := attributes["num_node_groups"]; ok {
			params.NumNodeGroups = aws.Int64(int64(v.(int)))
		}

		if v, ok := attributes["replicas_per_node_group"]; ok {
			params.ReplicasPerNodeGroup = aws.Int64(int64(v.(int)))
		}
	}

	if cacheClustersOk {
		params.NumCacheClusters = aws.Int64(int64(cacheClusters.(int)))
	}

	resp, err := conn.CreateReplicationGroup(params)
	if err != nil {
		return fmt.Errorf("Error creating Elasticache Replication Group: %s", err)
	}

	d.SetId(*resp.ReplicationGroup.ReplicationGroupId)

	pending := []string{"creating", "modifying", "restoring", "snapshotting"}
	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     []string{"available"},
		Refresh:    cacheReplicationGroupStateRefreshFunc(conn, d.Id(), "available", pending),
		Timeout:    d.Timeout(schema.TimeoutCreate),
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
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
		if isAWSErr(err, elasticache.ErrCodeReplicationGroupNotFoundFault, "") {
			log.Printf("[WARN] Elasticache Replication Group (%s) not found", d.Id())
			d.SetId("")
			return nil
		}

		return err
	}

	var rgp *elasticache.ReplicationGroup
	for _, r := range res.ReplicationGroups {
		if *r.ReplicationGroupId == d.Id() {
			rgp = r
		}
	}

	if rgp == nil {
		log.Printf("[WARN] Replication Group (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	if *rgp.Status == "deleting" {
		log.Printf("[WARN] The Replication Group %q is currently in the `deleting` state", d.Id())
		d.SetId("")
		return nil
	}

	if rgp.AutomaticFailover != nil {
		switch strings.ToLower(*rgp.AutomaticFailover) {
		case "disabled", "disabling":
			d.Set("automatic_failover_enabled", false)
		case "enabled", "enabling":
			d.Set("automatic_failover_enabled", true)
		default:
			log.Printf("Unknown AutomaticFailover state %s", *rgp.AutomaticFailover)
		}
	}

	d.Set("replication_group_description", rgp.Description)
	d.Set("number_cache_clusters", len(rgp.MemberClusters))
	if err := d.Set("member_clusters", flattenStringList(rgp.MemberClusters)); err != nil {
		return fmt.Errorf("error setting member_clusters: %s", err)
	}
	if err := d.Set("cluster_mode", flattenElasticacheNodeGroupsToClusterMode(aws.BoolValue(rgp.ClusterEnabled), rgp.NodeGroups)); err != nil {
		return fmt.Errorf("error setting cluster_mode attribute: %s", err)
	}
	d.Set("replication_group_id", rgp.ReplicationGroupId)

	if rgp.NodeGroups != nil {
		if len(rgp.NodeGroups[0].NodeGroupMembers) == 0 {
			return nil
		}

		cacheCluster := *rgp.NodeGroups[0].NodeGroupMembers[0]

		res, err := conn.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{
			CacheClusterId:    cacheCluster.CacheClusterId,
			ShowCacheNodeInfo: aws.Bool(true),
		})
		if err != nil {
			return err
		}

		if len(res.CacheClusters) == 0 {
			return nil
		}

		c := res.CacheClusters[0]
		d.Set("node_type", c.CacheNodeType)
		d.Set("engine", c.Engine)
		d.Set("engine_version", c.EngineVersion)
		d.Set("subnet_group_name", c.CacheSubnetGroupName)
		d.Set("security_group_names", flattenElastiCacheSecurityGroupNames(c.CacheSecurityGroups))
		d.Set("security_group_ids", flattenElastiCacheSecurityGroupIds(c.SecurityGroups))

		if c.CacheParameterGroup != nil {
			d.Set("parameter_group_name", c.CacheParameterGroup.CacheParameterGroupName)
		}

		d.Set("maintenance_window", c.PreferredMaintenanceWindow)
		d.Set("snapshot_window", rgp.SnapshotWindow)
		d.Set("snapshot_retention_limit", rgp.SnapshotRetentionLimit)

		if rgp.ConfigurationEndpoint != nil {
			d.Set("port", rgp.ConfigurationEndpoint.Port)
			d.Set("configuration_endpoint_address", rgp.ConfigurationEndpoint.Address)
		} else {
			d.Set("port", rgp.NodeGroups[0].PrimaryEndpoint.Port)
			d.Set("primary_endpoint_address", rgp.NodeGroups[0].PrimaryEndpoint.Address)
		}

		d.Set("auto_minor_version_upgrade", c.AutoMinorVersionUpgrade)
		d.Set("at_rest_encryption_enabled", c.AtRestEncryptionEnabled)
		d.Set("transit_encryption_enabled", c.TransitEncryptionEnabled)

		if c.AuthTokenEnabled != nil && !*c.AuthTokenEnabled {
			d.Set("auth_token", nil)
		}
	}

	return nil
}

func resourceAwsElasticacheReplicationGroupUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	if d.HasChange("cluster_mode.0.num_node_groups") {
		o, n := d.GetChange("cluster_mode.0.num_node_groups")
		oldNumNodeGroups := o.(int)
		newNumNodeGroups := n.(int)

		input := &elasticache.ModifyReplicationGroupShardConfigurationInput{
			ApplyImmediately:   aws.Bool(true),
			NodeGroupCount:     aws.Int64(int64(newNumNodeGroups)),
			ReplicationGroupId: aws.String(d.Id()),
		}

		if oldNumNodeGroups > newNumNodeGroups {
			// Node Group IDs are 1 indexed: 0001 through 0015
			// Loop from highest old ID until we reach highest new ID
			nodeGroupsToRemove := []string{}
			for i := oldNumNodeGroups; i > newNumNodeGroups; i-- {
				nodeGroupID := fmt.Sprintf("%04d", i)
				nodeGroupsToRemove = append(nodeGroupsToRemove, nodeGroupID)
			}
			input.NodeGroupsToRemove = aws.StringSlice(nodeGroupsToRemove)
		}

		log.Printf("[DEBUG] Modifying Elasticache Replication Group (%s) shard configuration: %s", d.Id(), input)
		_, err := conn.ModifyReplicationGroupShardConfiguration(input)
		if err != nil {
			return fmt.Errorf("error modifying Elasticache Replication Group shard configuration: %s", err)
		}

		err = waitForModifyElasticacheReplicationGroup(conn, d.Id(), d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return fmt.Errorf("error waiting for Elasticache Replication Group (%s) shard reconfiguration completion: %s", d.Id(), err)
		}
	}

	if d.HasChange("number_cache_clusters") {
		o, n := d.GetChange("number_cache_clusters")
		oldNumberCacheClusters := o.(int)
		newNumberCacheClusters := n.(int)

		// We will try to use similar naming to the console, which are 1 indexed: RGID-001 through RGID-006
		var addClusterIDs, removeClusterIDs []string
		for clusterID := oldNumberCacheClusters + 1; clusterID <= newNumberCacheClusters; clusterID++ {
			addClusterIDs = append(addClusterIDs, fmt.Sprintf("%s-%03d", d.Id(), clusterID))
		}
		for clusterID := oldNumberCacheClusters; clusterID >= (newNumberCacheClusters + 1); clusterID-- {
			removeClusterIDs = append(removeClusterIDs, fmt.Sprintf("%s-%03d", d.Id(), clusterID))
		}

		if len(addClusterIDs) > 0 {
			// Kick off all the Cache Cluster creations
			for _, cacheClusterID := range addClusterIDs {
				input := &elasticache.CreateCacheClusterInput{
					CacheClusterId:     aws.String(cacheClusterID),
					ReplicationGroupId: aws.String(d.Id()),
				}
				_, err := createElasticacheCacheCluster(conn, input)
				if err != nil {
					// Future enhancement: we could retry creation with random ID on naming collision
					// if isAWSErr(err, elasticache.ErrCodeCacheClusterAlreadyExistsFault, "") { ... }
					return fmt.Errorf("error creating Elasticache Cache Cluster (adding replica): %s", err)
				}
			}

			// Wait for all Cache Cluster creations
			for _, cacheClusterID := range addClusterIDs {
				err := waitForCreateElasticacheCacheCluster(conn, cacheClusterID, d.Timeout(schema.TimeoutUpdate))
				if err != nil {
					return fmt.Errorf("error waiting for Elasticache Cache Cluster (%s) to be created (adding replica): %s", cacheClusterID, err)
				}
			}
		}

		if len(removeClusterIDs) > 0 {
			// Cannot reassign primary cluster ID while automatic failover is enabled
			// If we temporarily disable automatic failover, ensure we re-enable it
			reEnableAutomaticFailover := false

			// Kick off all the Cache Cluster deletions
			for _, cacheClusterID := range removeClusterIDs {
				err := deleteElasticacheCacheCluster(conn, cacheClusterID)
				if err != nil {
					// Future enhancement: we could retry deletion with random existing ID on missing name
					// if isAWSErr(err, elasticache.ErrCodeCacheClusterNotFoundFault, "") { ... }
					if !isAWSErr(err, elasticache.ErrCodeInvalidCacheClusterStateFault, "serving as primary") {
						return fmt.Errorf("error deleting Elasticache Cache Cluster (%s) (removing replica): %s", cacheClusterID, err)
					}

					// Use Replication Group MemberClusters to find a new primary cache cluster ID
					// that is not in removeClusterIDs
					newPrimaryClusterID := ""

					describeReplicationGroupInput := &elasticache.DescribeReplicationGroupsInput{
						ReplicationGroupId: aws.String(d.Id()),
					}
					log.Printf("[DEBUG] Reading Elasticache Replication Group: %s", describeReplicationGroupInput)
					output, err := conn.DescribeReplicationGroups(describeReplicationGroupInput)
					if err != nil {
						return fmt.Errorf("error reading Elasticache Replication Group (%s) to determine new primary: %s", d.Id(), err)
					}
					if output == nil || len(output.ReplicationGroups) == 0 || len(output.ReplicationGroups[0].MemberClusters) == 0 {
						return fmt.Errorf("error reading Elasticache Replication Group (%s) to determine new primary: missing replication group information", d.Id())
					}

					for _, memberClusterPtr := range output.ReplicationGroups[0].MemberClusters {
						memberCluster := aws.StringValue(memberClusterPtr)
						memberClusterInRemoveClusterIDs := false
						for _, removeClusterID := range removeClusterIDs {
							if memberCluster == removeClusterID {
								memberClusterInRemoveClusterIDs = true
								break
							}
						}
						if !memberClusterInRemoveClusterIDs {
							newPrimaryClusterID = memberCluster
							break
						}
					}
					if newPrimaryClusterID == "" {
						return fmt.Errorf("error reading Elasticache Replication Group (%s) to determine new primary: unable to assign new primary", d.Id())
					}

					// Disable automatic failover if enabled
					// Must be applied previous to trying to set new primary
					// InvalidReplicationGroupState: Cannot manually promote a new master cache cluster while autofailover is enabled
					if aws.StringValue(output.ReplicationGroups[0].AutomaticFailover) == elasticache.AutomaticFailoverStatusEnabled {
						// Be kind and rewind
						if d.Get("automatic_failover_enabled").(bool) {
							reEnableAutomaticFailover = true
						}

						modifyReplicationGroupInput := &elasticache.ModifyReplicationGroupInput{
							ApplyImmediately:         aws.Bool(true),
							AutomaticFailoverEnabled: aws.Bool(false),
							ReplicationGroupId:       aws.String(d.Id()),
						}
						log.Printf("[DEBUG] Modifying Elasticache Replication Group: %s", modifyReplicationGroupInput)
						_, err = conn.ModifyReplicationGroup(modifyReplicationGroupInput)
						if err != nil {
							return fmt.Errorf("error modifying Elasticache Replication Group (%s) to set new primary: %s", d.Id(), err)
						}
						err = waitForModifyElasticacheReplicationGroup(conn, d.Id(), d.Timeout(schema.TimeoutUpdate))
						if err != nil {
							return fmt.Errorf("error waiting for Elasticache Replication Group (%s) to be available: %s", d.Id(), err)
						}
					}

					// Set new primary
					modifyReplicationGroupInput := &elasticache.ModifyReplicationGroupInput{
						ApplyImmediately:   aws.Bool(true),
						PrimaryClusterId:   aws.String(newPrimaryClusterID),
						ReplicationGroupId: aws.String(d.Id()),
					}
					log.Printf("[DEBUG] Modifying Elasticache Replication Group: %s", modifyReplicationGroupInput)
					_, err = conn.ModifyReplicationGroup(modifyReplicationGroupInput)
					if err != nil {
						return fmt.Errorf("error modifying Elasticache Replication Group (%s) to set new primary: %s", d.Id(), err)
					}
					err = waitForModifyElasticacheReplicationGroup(conn, d.Id(), d.Timeout(schema.TimeoutUpdate))
					if err != nil {
						return fmt.Errorf("error waiting for Elasticache Replication Group (%s) to be available: %s", d.Id(), err)
					}

					// Finally retry deleting the cache cluster
					err = deleteElasticacheCacheCluster(conn, cacheClusterID)
					if err != nil {
						return fmt.Errorf("error deleting Elasticache Cache Cluster (%s) (removing replica after setting new primary): %s", cacheClusterID, err)
					}
				}
			}

			// Wait for all Cache Cluster deletions
			for _, cacheClusterID := range removeClusterIDs {
				err := waitForDeleteElasticacheCacheCluster(conn, cacheClusterID, d.Timeout(schema.TimeoutUpdate))
				if err != nil {
					return fmt.Errorf("error waiting for Elasticache Cache Cluster (%s) to be deleted (removing replica): %s", cacheClusterID, err)
				}
			}

			// Re-enable automatic failover if we needed to temporarily disable it
			if reEnableAutomaticFailover {
				input := &elasticache.ModifyReplicationGroupInput{
					ApplyImmediately:         aws.Bool(true),
					AutomaticFailoverEnabled: aws.Bool(true),
					ReplicationGroupId:       aws.String(d.Id()),
				}
				log.Printf("[DEBUG] Modifying Elasticache Replication Group: %s", input)
				_, err := conn.ModifyReplicationGroup(input)
				if err != nil {
					return fmt.Errorf("error modifying Elasticache Replication Group (%s) to re-enable automatic failover: %s", d.Id(), err)
				}
			}
		}
	}

	requestUpdate := false
	params := &elasticache.ModifyReplicationGroupInput{
		ApplyImmediately:   aws.Bool(d.Get("apply_immediately").(bool)),
		ReplicationGroupId: aws.String(d.Id()),
	}

	if d.HasChange("replication_group_description") {
		params.ReplicationGroupDescription = aws.String(d.Get("replication_group_description").(string))
		requestUpdate = true
	}

	if d.HasChange("automatic_failover_enabled") {
		params.AutomaticFailoverEnabled = aws.Bool(d.Get("automatic_failover_enabled").(bool))
		requestUpdate = true
	}

	if d.HasChange("auto_minor_version_upgrade") {
		params.AutoMinorVersionUpgrade = aws.Bool(d.Get("auto_minor_version_upgrade").(bool))
		requestUpdate = true
	}

	if d.HasChange("security_group_ids") {
		if attr := d.Get("security_group_ids").(*schema.Set); attr.Len() > 0 {
			params.SecurityGroupIds = expandStringList(attr.List())
			requestUpdate = true
		}
	}

	if d.HasChange("security_group_names") {
		if attr := d.Get("security_group_names").(*schema.Set); attr.Len() > 0 {
			params.CacheSecurityGroupNames = expandStringList(attr.List())
			requestUpdate = true
		}
	}

	if d.HasChange("maintenance_window") {
		params.PreferredMaintenanceWindow = aws.String(d.Get("maintenance_window").(string))
		requestUpdate = true
	}

	if d.HasChange("notification_topic_arn") {
		params.NotificationTopicArn = aws.String(d.Get("notification_topic_arn").(string))
		requestUpdate = true
	}

	if d.HasChange("parameter_group_name") {
		params.CacheParameterGroupName = aws.String(d.Get("parameter_group_name").(string))
		requestUpdate = true
	}

	if d.HasChange("engine_version") {
		params.EngineVersion = aws.String(d.Get("engine_version").(string))
		requestUpdate = true
	}

	if d.HasChange("snapshot_retention_limit") {
		// This is a real hack to set the Snapshotting Cluster ID to be the first Cluster in the RG
		o, _ := d.GetChange("snapshot_retention_limit")
		if o.(int) == 0 {
			params.SnapshottingClusterId = aws.String(fmt.Sprintf("%s-001", d.Id()))
		}

		params.SnapshotRetentionLimit = aws.Int64(int64(d.Get("snapshot_retention_limit").(int)))
		requestUpdate = true
	}

	if d.HasChange("snapshot_window") {
		params.SnapshotWindow = aws.String(d.Get("snapshot_window").(string))
		requestUpdate = true
	}

	if d.HasChange("node_type") {
		params.CacheNodeType = aws.String(d.Get("node_type").(string))
		requestUpdate = true
	}

	if requestUpdate {
		_, err := conn.ModifyReplicationGroup(params)
		if err != nil {
			return fmt.Errorf("error updating Elasticache Replication Group (%s): %s", d.Id(), err)
		}

		err = waitForModifyElasticacheReplicationGroup(conn, d.Id(), d.Timeout(schema.TimeoutUpdate))
		if err != nil {
			return fmt.Errorf("error waiting for Elasticache Replication Group (%s) to be updated: %s", d.Id(), err)
		}
	}
	return resourceAwsElasticacheReplicationGroupRead(d, meta)
}

func resourceAwsElasticacheReplicationGroupDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	err := deleteElasticacheReplicationGroup(d.Id(), 40*time.Minute, conn)
	if err != nil {
		return fmt.Errorf("error deleting Elasticache Replication Group (%s): %s", d.Id(), err)
	}

	return nil
}

func cacheReplicationGroupStateRefreshFunc(conn *elasticache.ElastiCache, replicationGroupId, givenState string, pending []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeReplicationGroups(&elasticache.DescribeReplicationGroupsInput{
			ReplicationGroupId: aws.String(replicationGroupId),
		})
		if err != nil {
			if isAWSErr(err, elasticache.ErrCodeReplicationGroupNotFoundFault, "") {
				log.Printf("[DEBUG] Replication Group Not Found")
				return nil, "", nil
			}

			log.Printf("[ERROR] cacheClusterReplicationGroupStateRefreshFunc: %s", err)
			return nil, "", err
		}

		if len(resp.ReplicationGroups) == 0 {
			return nil, "", fmt.Errorf("Error: no Cache Replication Groups found for id (%s)", replicationGroupId)
		}

		var rg *elasticache.ReplicationGroup
		for _, replicationGroup := range resp.ReplicationGroups {
			if *replicationGroup.ReplicationGroupId == replicationGroupId {
				log.Printf("[DEBUG] Found matching ElastiCache Replication Group: %s", *replicationGroup.ReplicationGroupId)
				rg = replicationGroup
			}
		}

		if rg == nil {
			return nil, "", fmt.Errorf("Error: no matching ElastiCache Replication Group for id (%s)", replicationGroupId)
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

		return rg, *rg.Status, nil
	}
}

func deleteElasticacheReplicationGroup(replicationGroupID string, timeout time.Duration, conn *elasticache.ElastiCache) error {
	input := &elasticache.DeleteReplicationGroupInput{
		ReplicationGroupId: aws.String(replicationGroupID),
	}

	// 10 minutes should give any creating/deleting cache clusters or snapshots time to complete
	err := resource.Retry(10*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteReplicationGroup(input)
		if err != nil {
			if isAWSErr(err, elasticache.ErrCodeReplicationGroupNotFoundFault, "") {
				return nil
			}
			// Cache Cluster is creating/deleting or Replication Group is snapshotting
			// InvalidReplicationGroupState: Cache cluster tf-acc-test-uqhe-003 is not in a valid state to be deleted
			if isAWSErr(err, elasticache.ErrCodeInvalidReplicationGroupStateFault, "") {
				return resource.RetryableError(err)
			}
			return resource.NonRetryableError(err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Waiting for deletion: %s", replicationGroupID)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "available", "deleting"},
		Target:     []string{},
		Refresh:    cacheReplicationGroupStateRefreshFunc(conn, replicationGroupID, "", []string{}),
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	_, err = stateConf.WaitForState()
	return err
}

func flattenElasticacheNodeGroupsToClusterMode(clusterEnabled bool, nodeGroups []*elasticache.NodeGroup) []map[string]interface{} {
	if !clusterEnabled {
		return []map[string]interface{}{}
	}

	m := map[string]interface{}{
		"num_node_groups":         0,
		"replicas_per_node_group": 0,
	}

	if len(nodeGroups) == 0 {
		return []map[string]interface{}{m}
	}

	m["num_node_groups"] = len(nodeGroups)
	m["replicas_per_node_group"] = (len(nodeGroups[0].NodeGroupMembers) - 1)
	return []map[string]interface{}{m}
}

func waitForModifyElasticacheReplicationGroup(conn *elasticache.ElastiCache, replicationGroupID string, timeout time.Duration) error {
	pending := []string{"creating", "modifying", "snapshotting"}
	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     []string{"available"},
		Refresh:    cacheReplicationGroupStateRefreshFunc(conn, replicationGroupID, "available", pending),
		Timeout:    timeout,
		MinTimeout: 10 * time.Second,
		Delay:      30 * time.Second,
	}

	log.Printf("[DEBUG] Waiting for Elasticache Replication Group (%s) to become available", replicationGroupID)
	_, err := stateConf.WaitForState()
	return err
}

func validateAwsElastiCacheReplicationGroupEngine(v interface{}, k string) (ws []string, errors []error) {
	if strings.ToLower(v.(string)) != "redis" {
		errors = append(errors, fmt.Errorf("The only acceptable Engine type when using Replication Groups is Redis"))
	}
	return
}

func validateAwsElastiCacheReplicationGroupId(v interface{}, k string) (ws []string, errors []error) {
	value := v.(string)
	if (len(value) < 1) || (len(value) > 20) {
		errors = append(errors, fmt.Errorf(
			"%q must contain from 1 to 20 alphanumeric characters or hyphens", k))
	}
	if !regexp.MustCompile(`^[0-9a-zA-Z-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"only alphanumeric characters and hyphens allowed in %q", k))
	}
	if !regexp.MustCompile(`^[a-zA-Z]`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"first character of %q must be a letter", k))
	}
	if regexp.MustCompile(`--`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot contain two consecutive hyphens", k))
	}
	if regexp.MustCompile(`-$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"%q cannot end with a hyphen", k))
	}
	return
}
