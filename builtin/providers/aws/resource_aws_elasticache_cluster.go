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
	"github.com/hashicorp/terraform/helper/hashcode"
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
			},
			"engine": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"node_type": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"num_cache_nodes": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},
			"parameter_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
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
				ForceNew: true,
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
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
			},
			"security_group_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set: func(v interface{}) int {
					return hashcode.String(v.(string))
				},
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
					},
				},
			},

			"tags": tagsSchema(),
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
		CacheClusterID:          aws.String(clusterId),
		CacheNodeType:           aws.String(nodeType),
		NumCacheNodes:           aws.Long(numNodes),
		Engine:                  aws.String(engine),
		EngineVersion:           aws.String(engineVersion),
		Port:                    aws.Long(port),
		CacheSubnetGroupName:    aws.String(subnetGroupName),
		CacheSecurityGroupNames: securityNames,
		SecurityGroupIDs:        securityIds,
		Tags:                    tags,
	}

	// parameter groups are optional and can be defaulted by AWS
	if v, ok := d.GetOk("parameter_group_name"); ok {
		req.CacheParameterGroupName = aws.String(v.(string))
	}

	_, err := conn.CreateCacheCluster(req)
	if err != nil {
		return fmt.Errorf("Error creating Elasticache: %s", err)
	}

	pending := []string{"creating"}
	stateConf := &resource.StateChangeConf{
		Pending:    pending,
		Target:     "available",
		Refresh:    CacheClusterStateRefreshFunc(conn, d.Id(), "available", pending),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	log.Printf("[DEBUG] Waiting for state to become available: %v", d.Id())
	_, sterr := stateConf.WaitForState()
	if sterr != nil {
		return fmt.Errorf("Error waiting for elasticache (%s) to be created: %s", d.Id(), sterr)
	}

	d.SetId(clusterId)

	return resourceAwsElasticacheClusterRead(d, meta)
}

func resourceAwsElasticacheClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn
	req := &elasticache.DescribeCacheClustersInput{
		CacheClusterID:    aws.String(d.Id()),
		ShowCacheNodeInfo: aws.Boolean(true),
	}

	res, err := conn.DescribeCacheClusters(req)
	if err != nil {
		return err
	}

	if len(res.CacheClusters) == 1 {
		c := res.CacheClusters[0]
		d.Set("cluster_id", c.CacheClusterID)
		d.Set("node_type", c.CacheNodeType)
		d.Set("num_cache_nodes", c.NumCacheNodes)
		d.Set("engine", c.Engine)
		d.Set("engine_version", c.EngineVersion)
		if c.ConfigurationEndpoint != nil {
			d.Set("port", c.ConfigurationEndpoint.Port)
		}
		d.Set("subnet_group_name", c.CacheSubnetGroupName)
		d.Set("security_group_names", c.CacheSecurityGroups)
		d.Set("security_group_ids", c.SecurityGroups)
		d.Set("parameter_group_name", c.CacheParameterGroup)

		if err := setCacheNodeData(d, c); err != nil {
			return err
		}
		// list tags for resource
		// set tags
		arn, err := buildECARN(d, meta)
		if err != nil {
			log.Printf("[DEBUG] Error building ARN for ElastiCache Cluster, not setting Tags for cluster %s", *c.CacheClusterID)
		} else {
			resp, err := conn.ListTagsForResource(&elasticache.ListTagsForResourceInput{
				ResourceName: aws.String(arn),
			})

			if err != nil {
				log.Printf("[DEBUG] Error retreiving tags for ARN: %s", arn)
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
	return resourceAwsElasticacheClusterRead(d, meta)
}

func setCacheNodeData(d *schema.ResourceData, c *elasticache.CacheCluster) error {
	sortedCacheNodes := make([]*elasticache.CacheNode, len(c.CacheNodes))
	copy(sortedCacheNodes, c.CacheNodes)
	sort.Sort(byCacheNodeId(sortedCacheNodes))

	cacheNodeData := make([]map[string]interface{}, 0, len(sortedCacheNodes))

	for _, node := range sortedCacheNodes {
		if node.CacheNodeID == nil || node.Endpoint == nil || node.Endpoint.Address == nil || node.Endpoint.Port == nil {
			return fmt.Errorf("Unexpected nil pointer in: %#v", node)
		}
		cacheNodeData = append(cacheNodeData, map[string]interface{}{
			"id":      *node.CacheNodeID,
			"address": *node.Endpoint.Address,
			"port":    int(*node.Endpoint.Port),
		})
	}

	return d.Set("cache_nodes", cacheNodeData)
}

type byCacheNodeId []*elasticache.CacheNode

func (b byCacheNodeId) Len() int      { return len(b) }
func (b byCacheNodeId) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byCacheNodeId) Less(i, j int) bool {
	return b[i].CacheNodeID != nil && b[j].CacheNodeID != nil &&
		*b[i].CacheNodeID < *b[j].CacheNodeID
}

func resourceAwsElasticacheClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).elasticacheconn

	req := &elasticache.DeleteCacheClusterInput{
		CacheClusterID: aws.String(d.Id()),
	}
	_, err := conn.DeleteCacheCluster(req)
	if err != nil {
		return err
	}

	log.Printf("[DEBUG] Waiting for deletion: %v", d.Id())
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "available", "deleting", "incompatible-parameters", "incompatible-network", "restore-failed"},
		Target:     "",
		Refresh:    CacheClusterStateRefreshFunc(conn, d.Id(), "", []string{}),
		Timeout:    10 * time.Minute,
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

func CacheClusterStateRefreshFunc(conn *elasticache.ElastiCache, clusterID, givenState string, pending []string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		resp, err := conn.DescribeCacheClusters(&elasticache.DescribeCacheClustersInput{
			CacheClusterID:    aws.String(clusterID),
			ShowCacheNodeInfo: aws.Boolean(true),
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

		c := resp.CacheClusters[0]
		log.Printf("[DEBUG] status: %v", *c.CacheClusterStatus)

		// return the current state if it's in the pending array
		for _, p := range pending {
			s := *c.CacheClusterStatus
			if p == s {
				log.Printf("[DEBUG] Return with status: %v", *c.CacheClusterStatus)
				return c, p, nil
			}
		}

		// return given state if it's not in pending
		if givenState != "" {
			// check to make sure we have the node count we're expecting
			if int64(len(c.CacheNodes)) != *c.NumCacheNodes {
				log.Printf("[DEBUG] Node count is not what is expected: %d found, %d expected", len(c.CacheNodes), *c.NumCacheNodes)
				return nil, "creating", nil
			}
			// loop the nodes and check their status as well
			for _, n := range c.CacheNodes {
				if n.CacheNodeStatus != nil && *n.CacheNodeStatus != "available" {
					log.Printf("[DEBUG] Node (%s) is not yet available, status: %s", *n.CacheNodeID, *n.CacheNodeStatus)
					return nil, "creating", nil
				}
			}
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
	userARN := *resp.User.ARN
	accountID := strings.Split(userARN, ":")[4]
	arn := fmt.Sprintf("arn:aws:elasticache:%s:%s:cluster:%s", region, accountID, d.Id())
	return arn, nil
}
