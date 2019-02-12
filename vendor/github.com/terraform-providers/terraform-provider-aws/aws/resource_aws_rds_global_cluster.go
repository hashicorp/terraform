package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsRDSGlobalCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRDSGlobalClusterCreate,
		Read:   resourceAwsRDSGlobalClusterRead,
		Update: resourceAwsRDSGlobalClusterUpdate,
		Delete: resourceAwsRDSGlobalClusterDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"database_name": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
			"deletion_protection": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},
			"engine": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "aurora",
				ValidateFunc: validation.StringInSlice([]string{
					"aurora",
				}, false),
			},
			"engine_version": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},
			"global_cluster_identifier": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"global_cluster_resource_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"storage_encrypted": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func resourceAwsRDSGlobalClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	input := &rds.CreateGlobalClusterInput{
		DeletionProtection:      aws.Bool(d.Get("deletion_protection").(bool)),
		GlobalClusterIdentifier: aws.String(d.Get("global_cluster_identifier").(string)),
		StorageEncrypted:        aws.Bool(d.Get("storage_encrypted").(bool)),
	}

	if v, ok := d.GetOk("database_name"); ok {
		input.DatabaseName = aws.String(v.(string))
	}

	if v, ok := d.GetOk("engine"); ok {
		input.Engine = aws.String(v.(string))
	}

	if v, ok := d.GetOk("engine_version"); ok {
		input.EngineVersion = aws.String(v.(string))
	}

	log.Printf("[DEBUG] Creating RDS Global Cluster: %s", input)
	output, err := conn.CreateGlobalCluster(input)
	if err != nil {
		return fmt.Errorf("error creating RDS Global Cluster: %s", err)
	}

	d.SetId(aws.StringValue(output.GlobalCluster.GlobalClusterIdentifier))

	if err := waitForRdsGlobalClusterCreation(conn, d.Id()); err != nil {
		return fmt.Errorf("error waiting for RDS Global Cluster (%s) availability: %s", d.Id(), err)
	}

	return resourceAwsRDSGlobalClusterRead(d, meta)
}

func resourceAwsRDSGlobalClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	globalCluster, err := rdsDescribeGlobalCluster(conn, d.Id())

	if isAWSErr(err, rds.ErrCodeGlobalClusterNotFoundFault, "") {
		log.Printf("[WARN] RDS Global Cluster (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if err != nil {
		return fmt.Errorf("error reading RDS Global Cluster: %s", err)
	}

	if globalCluster == nil {
		log.Printf("[WARN] RDS Global Cluster (%s) not found, removing from state", d.Id())
		d.SetId("")
		return nil
	}

	if aws.StringValue(globalCluster.Status) == "deleting" || aws.StringValue(globalCluster.Status) == "deleted" {
		log.Printf("[WARN] RDS Global Cluster (%s) in deleted state (%s), removing from state", d.Id(), aws.StringValue(globalCluster.Status))
		d.SetId("")
		return nil
	}

	d.Set("arn", globalCluster.GlobalClusterArn)
	d.Set("database_name", globalCluster.DatabaseName)
	d.Set("deletion_protection", globalCluster.DeletionProtection)
	d.Set("engine", globalCluster.Engine)
	d.Set("engine_version", globalCluster.EngineVersion)
	d.Set("global_cluster_identifier", globalCluster.GlobalClusterIdentifier)
	d.Set("global_cluster_resource_id", globalCluster.GlobalClusterResourceId)
	d.Set("storage_encrypted", globalCluster.StorageEncrypted)

	return nil
}

func resourceAwsRDSGlobalClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	input := &rds.ModifyGlobalClusterInput{
		DeletionProtection:      aws.Bool(d.Get("deletion_protection").(bool)),
		GlobalClusterIdentifier: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Updating RDS Global Cluster (%s): %s", d.Id(), input)
	_, err := conn.ModifyGlobalCluster(input)

	if isAWSErr(err, rds.ErrCodeGlobalClusterNotFoundFault, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting RDS Global Cluster: %s", err)
	}

	if err := waitForRdsGlobalClusterUpdate(conn, d.Id()); err != nil {
		return fmt.Errorf("error waiting for RDS Global Cluster (%s) update: %s", d.Id(), err)
	}

	return nil
}

func resourceAwsRDSGlobalClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	input := &rds.DeleteGlobalClusterInput{
		GlobalClusterIdentifier: aws.String(d.Id()),
	}

	log.Printf("[DEBUG] Deleting RDS Global Cluster (%s): %s", d.Id(), input)

	// Allow for eventual consistency
	// InvalidGlobalClusterStateFault: Global Cluster arn:aws:rds::123456789012:global-cluster:tf-acc-test-5618525093076697001-0 is not empty
	err := resource.Retry(1*time.Minute, func() *resource.RetryError {
		_, err := conn.DeleteGlobalCluster(input)

		if isAWSErr(err, rds.ErrCodeInvalidGlobalClusterStateFault, "is not empty") {
			return resource.RetryableError(err)
		}

		if err != nil {
			return resource.NonRetryableError(err)
		}

		return nil
	})

	if isAWSErr(err, rds.ErrCodeGlobalClusterNotFoundFault, "") {
		return nil
	}

	if err != nil {
		return fmt.Errorf("error deleting RDS Global Cluster: %s", err)
	}

	if err := waitForRdsGlobalClusterDeletion(conn, d.Id()); err != nil {
		return fmt.Errorf("error waiting for RDS Global Cluster (%s) deletion: %s", d.Id(), err)
	}

	return nil
}

func rdsDescribeGlobalCluster(conn *rds.RDS, globalClusterID string) (*rds.GlobalCluster, error) {
	var globalCluster *rds.GlobalCluster

	input := &rds.DescribeGlobalClustersInput{
		GlobalClusterIdentifier: aws.String(globalClusterID),
	}

	log.Printf("[DEBUG] Reading RDS Global Cluster (%s): %s", globalClusterID, input)
	err := conn.DescribeGlobalClustersPages(input, func(page *rds.DescribeGlobalClustersOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, gc := range page.GlobalClusters {
			if gc == nil {
				continue
			}

			if aws.StringValue(gc.GlobalClusterIdentifier) == globalClusterID {
				globalCluster = gc
				return false
			}
		}

		return !lastPage
	})

	return globalCluster, err
}

func rdsDescribeGlobalClusterFromDbClusterARN(conn *rds.RDS, dbClusterARN string) (*rds.GlobalCluster, error) {
	var globalCluster *rds.GlobalCluster

	input := &rds.DescribeGlobalClustersInput{
		Filters: []*rds.Filter{
			{
				Name:   aws.String("db-cluster-id"),
				Values: []*string{aws.String(dbClusterARN)},
			},
		},
	}

	log.Printf("[DEBUG] Reading RDS Global Clusters: %s", input)
	err := conn.DescribeGlobalClustersPages(input, func(page *rds.DescribeGlobalClustersOutput, lastPage bool) bool {
		if page == nil {
			return !lastPage
		}

		for _, gc := range page.GlobalClusters {
			if gc == nil {
				continue
			}

			for _, globalClusterMember := range gc.GlobalClusterMembers {
				if aws.StringValue(globalClusterMember.DBClusterArn) == dbClusterARN {
					globalCluster = gc
					return false
				}
			}
		}

		return !lastPage
	})

	return globalCluster, err
}

func rdsGlobalClusterRefreshFunc(conn *rds.RDS, globalClusterID string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		globalCluster, err := rdsDescribeGlobalCluster(conn, globalClusterID)

		if isAWSErr(err, rds.ErrCodeGlobalClusterNotFoundFault, "") {
			return nil, "deleted", nil
		}

		if err != nil {
			return nil, "", fmt.Errorf("error reading RDS Global Cluster (%s): %s", globalClusterID, err)
		}

		if globalCluster == nil {
			return nil, "deleted", nil
		}

		return globalCluster, aws.StringValue(globalCluster.Status), nil
	}
}

func waitForRdsGlobalClusterCreation(conn *rds.RDS, globalClusterID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{"creating"},
		Target:  []string{"available"},
		Refresh: rdsGlobalClusterRefreshFunc(conn, globalClusterID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for RDS Global Cluster (%s) availability", globalClusterID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForRdsGlobalClusterUpdate(conn *rds.RDS, globalClusterID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{"modifying"},
		Target:  []string{"available"},
		Refresh: rdsGlobalClusterRefreshFunc(conn, globalClusterID),
		Timeout: 10 * time.Minute,
	}

	log.Printf("[DEBUG] Waiting for RDS Global Cluster (%s) availability", globalClusterID)
	_, err := stateConf.WaitForState()

	return err
}

func waitForRdsGlobalClusterDeletion(conn *rds.RDS, globalClusterID string) error {
	stateConf := &resource.StateChangeConf{
		Pending: []string{
			"available",
			"deleting",
		},
		Target:         []string{"deleted"},
		Refresh:        rdsGlobalClusterRefreshFunc(conn, globalClusterID),
		Timeout:        10 * time.Minute,
		NotFoundChecks: 1,
	}

	log.Printf("[DEBUG] Waiting for RDS Global Cluster (%s) deletion", globalClusterID)
	_, err := stateConf.WaitForState()

	if isResourceNotFoundError(err) {
		return nil
	}

	return err
}
