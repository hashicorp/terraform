package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsRDSClusterInstance() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRDSClusterInstanceCreate,
		Read:   resourceAwsRDSClusterInstanceRead,
		Update: resourceAwsRDSClusterInstanceUpdate,
		Delete: resourceAwsRDSClusterInstanceDelete,

		Schema: map[string]*schema.Schema{
			"identifier": &schema.Schema{
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validateRdsId,
			},

			"db_subnet_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"writer": &schema.Schema{
				Type:     schema.TypeBool,
				Computed: true,
			},

			"cluster_identifier": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"publicly_accessible": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"instance_class": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"tags": tagsSchema(),
		},
	}
}

func resourceAwsRDSClusterInstanceCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	tags := tagsFromMapRDS(d.Get("tags").(map[string]interface{}))

	createOpts := &rds.CreateDBInstanceInput{
		DBInstanceClass:     aws.String(d.Get("instance_class").(string)),
		DBClusterIdentifier: aws.String(d.Get("cluster_identifier").(string)),
		Engine:              aws.String("aurora"),
		PubliclyAccessible:  aws.Bool(d.Get("publicly_accessible").(bool)),
		Tags:                tags,
	}

	if v := d.Get("identifier").(string); v != "" {
		createOpts.DBInstanceIdentifier = aws.String(v)
	} else {
		createOpts.DBInstanceIdentifier = aws.String(resource.UniqueId())
	}

	if attr, ok := d.GetOk("db_subnet_group_name"); ok {
		createOpts.DBSubnetGroupName = aws.String(attr.(string))
	}

	log.Printf("[DEBUG] Creating RDS DB Instance opts: %s", createOpts)
	resp, err := conn.CreateDBInstance(createOpts)
	if err != nil {
		return err
	}

	d.SetId(*resp.DBInstance.DBInstanceIdentifier)

	// reuse db_instance refresh func
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "backing-up", "modifying"},
		Target:     []string{"available"},
		Refresh:    resourceAwsDbInstanceStateRefreshFunc(d, meta),
		Timeout:    40 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      10 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return err
	}

	return resourceAwsRDSClusterInstanceRead(d, meta)
}

func resourceAwsRDSClusterInstanceRead(d *schema.ResourceData, meta interface{}) error {
	db, err := resourceAwsDbInstanceRetrieve(d, meta)
	// Errors from this helper are always reportable
	if err != nil {
		return fmt.Errorf("[WARN] Error on retrieving RDS Cluster Instance (%s): %s", d.Id(), err)
	}
	// A nil response means "not found"
	if db == nil {
		log.Printf("[WARN] RDS Cluster Instance (%s): not found, removing from state.", d.Id())
		d.SetId("")
		return nil
	}

	// Retreive DB Cluster information, to determine if this Instance is a writer
	conn := meta.(*AWSClient).rdsconn
	resp, err := conn.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: db.DBClusterIdentifier,
	})

	var dbc *rds.DBCluster
	for _, c := range resp.DBClusters {
		if *c.DBClusterIdentifier == *db.DBClusterIdentifier {
			dbc = c
		}
	}

	if dbc == nil {
		return fmt.Errorf("[WARN] Error finding RDS Cluster (%s) for Cluster Instance (%s): %s",
			*db.DBClusterIdentifier, *db.DBInstanceIdentifier, err)
	}

	for _, m := range dbc.DBClusterMembers {
		if *db.DBInstanceIdentifier == *m.DBInstanceIdentifier {
			if *m.IsClusterWriter == true {
				d.Set("writer", true)
			} else {
				d.Set("writer", false)
			}
		}
	}

	if db.Endpoint != nil {
		d.Set("endpoint", db.Endpoint.Address)
		d.Set("port", db.Endpoint.Port)
	}

	d.Set("publicly_accessible", db.PubliclyAccessible)

	// Fetch and save tags
	arn, err := buildRDSARN(d.Id(), meta)
	if err != nil {
		log.Printf("[DEBUG] Error building ARN for RDS Cluster Instance (%s), not setting Tags", *db.DBInstanceIdentifier)
	} else {
		if err := saveTagsRDS(conn, d, arn); err != nil {
			log.Printf("[WARN] Failed to save tags for RDS Cluster Instance (%s): %s", *db.DBClusterIdentifier, err)
		}
	}

	return nil
}

func resourceAwsRDSClusterInstanceUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	if arn, err := buildRDSARN(d.Id(), meta); err == nil {
		if err := setTagsRDS(conn, d, arn); err != nil {
			return err
		}
	}

	return resourceAwsRDSClusterInstanceRead(d, meta)
}

func resourceAwsRDSClusterInstanceDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	log.Printf("[DEBUG] RDS Cluster Instance destroy: %v", d.Id())

	opts := rds.DeleteDBInstanceInput{DBInstanceIdentifier: aws.String(d.Id())}

	log.Printf("[DEBUG] RDS Cluster Instance destroy configuration: %s", opts)
	if _, err := conn.DeleteDBInstance(&opts); err != nil {
		return err
	}

	// re-uses db_instance refresh func
	log.Println("[INFO] Waiting for RDS Cluster Instance to be destroyed")
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"modifying", "deleting"},
		Target:     []string{},
		Refresh:    resourceAwsDbInstanceStateRefreshFunc(d, meta),
		Timeout:    40 * time.Minute,
		MinTimeout: 10 * time.Second,
	}

	if _, err := stateConf.WaitForState(); err != nil {
		return err
	}

	return nil

}
