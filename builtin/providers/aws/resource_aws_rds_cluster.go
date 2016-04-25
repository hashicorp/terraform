package aws

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceAwsRDSCluster() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsRDSClusterCreate,
		Read:   resourceAwsRDSClusterRead,
		Update: resourceAwsRDSClusterUpdate,
		Delete: resourceAwsRDSClusterDelete,

		Schema: map[string]*schema.Schema{

			"availability_zones": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				ForceNew: true,
				Computed: true,
				Set:      schema.HashString,
			},

			"cluster_identifier": &schema.Schema{
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validateRdsId,
			},

			"cluster_members": &schema.Schema{
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
				Computed: true,
				Set:      schema.HashString,
			},

			"database_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: true,
			},

			"db_subnet_group_name": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Computed: true,
			},

			"endpoint": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"engine": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"storage_encrypted": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				ForceNew: true,
			},

			"final_snapshot_identifier": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !regexp.MustCompile(`^[0-9A-Za-z-]+$`).MatchString(value) {
						es = append(es, fmt.Errorf(
							"only alphanumeric characters and hyphens allowed in %q", k))
					}
					if regexp.MustCompile(`--`).MatchString(value) {
						es = append(es, fmt.Errorf("%q cannot contain two consecutive hyphens", k))
					}
					if regexp.MustCompile(`-$`).MatchString(value) {
						es = append(es, fmt.Errorf("%q cannot end in a hyphen", k))
					}
					return
				},
			},

			"master_username": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"master_password": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},

			"port": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
			},

			// apply_immediately is used to determine when the update modifications
			// take place.
			// See http://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.Modifying.html
			"apply_immediately": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Computed: true,
			},

			"vpc_security_group_ids": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"preferred_backup_window": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},

			"preferred_maintenance_window": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				StateFunc: func(val interface{}) string {
					if val == nil {
						return ""
					}
					return strings.ToLower(val.(string))
				},
			},

			"backup_retention_period": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  1,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(int)
					if value > 35 {
						es = append(es, fmt.Errorf(
							"backup retention period cannot be more than 35 days"))
					}
					return
				},
			},
		},
	}
}

func resourceAwsRDSClusterCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	createOpts := &rds.CreateDBClusterInput{
		DBClusterIdentifier: aws.String(d.Get("cluster_identifier").(string)),
		Engine:              aws.String("aurora"),
		MasterUserPassword:  aws.String(d.Get("master_password").(string)),
		MasterUsername:      aws.String(d.Get("master_username").(string)),
		StorageEncrypted:    aws.Bool(d.Get("storage_encrypted").(bool)),
	}

	if v := d.Get("database_name"); v.(string) != "" {
		createOpts.DatabaseName = aws.String(v.(string))
	}

	if attr, ok := d.GetOk("port"); ok {
		createOpts.Port = aws.Int64(int64(attr.(int)))
	}

	if attr, ok := d.GetOk("db_subnet_group_name"); ok {
		createOpts.DBSubnetGroupName = aws.String(attr.(string))
	}

	if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
		createOpts.VpcSecurityGroupIds = expandStringList(attr.List())
	}

	if attr := d.Get("availability_zones").(*schema.Set); attr.Len() > 0 {
		createOpts.AvailabilityZones = expandStringList(attr.List())
	}

	if v, ok := d.GetOk("backup_retention_period"); ok {
		createOpts.BackupRetentionPeriod = aws.Int64(int64(v.(int)))
	}

	if v, ok := d.GetOk("preferred_backup_window"); ok {
		createOpts.PreferredBackupWindow = aws.String(v.(string))
	}

	if v, ok := d.GetOk("preferred_maintenance_window"); ok {
		createOpts.PreferredMaintenanceWindow = aws.String(v.(string))
	}

	log.Printf("[DEBUG] RDS Cluster create options: %s", createOpts)
	resp, err := conn.CreateDBCluster(createOpts)
	if err != nil {
		log.Printf("[ERROR] Error creating RDS Cluster: %s", err)
		return err
	}

	log.Printf("[DEBUG]: Cluster create response: %s", resp)
	d.SetId(*resp.DBCluster.DBClusterIdentifier)
	stateConf := &resource.StateChangeConf{
		Pending:    []string{"creating", "backing-up", "modifying"},
		Target:     []string{"available"},
		Refresh:    resourceAwsRDSClusterStateRefreshFunc(d, meta),
		Timeout:    5 * time.Minute,
		MinTimeout: 3 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("[WARN] Error waiting for RDS Cluster state to be \"available\": %s", err)
	}

	return resourceAwsRDSClusterRead(d, meta)
}

func resourceAwsRDSClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	resp, err := conn.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(d.Id()),
	})

	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if "DBClusterNotFoundFault" == awsErr.Code() {
				d.SetId("")
				log.Printf("[DEBUG] RDS Cluster (%s) not found", d.Id())
				return nil
			}
		}
		log.Printf("[DEBUG] Error describing RDS Cluster (%s)", d.Id())
		return err
	}

	var dbc *rds.DBCluster
	for _, c := range resp.DBClusters {
		if *c.DBClusterIdentifier == d.Id() {
			dbc = c
		}
	}

	if dbc == nil {
		log.Printf("[WARN] RDS Cluster (%s) not found", d.Id())
		d.SetId("")
		return nil
	}

	if err := d.Set("availability_zones", aws.StringValueSlice(dbc.AvailabilityZones)); err != nil {
		return fmt.Errorf("[DEBUG] Error saving AvailabilityZones to state for RDS Cluster (%s): %s", d.Id(), err)
	}

	// Only set the DatabaseName if it is not nil. There is a known API bug where
	// RDS accepts a DatabaseName but does not return it, causing a perpetual
	// diff.
	//	See https://github.com/hashicorp/terraform/issues/4671 for backstory
	if dbc.DatabaseName != nil {
		d.Set("database_name", dbc.DatabaseName)
	}

	d.Set("db_subnet_group_name", dbc.DBSubnetGroup)
	d.Set("endpoint", dbc.Endpoint)
	d.Set("engine", dbc.Engine)
	d.Set("master_username", dbc.MasterUsername)
	d.Set("port", dbc.Port)
	d.Set("storage_encrypted", dbc.StorageEncrypted)
	d.Set("backup_retention_period", dbc.BackupRetentionPeriod)
	d.Set("preferred_backup_window", dbc.PreferredBackupWindow)
	d.Set("preferred_maintenance_window", dbc.PreferredMaintenanceWindow)

	var vpcg []string
	for _, g := range dbc.VpcSecurityGroups {
		vpcg = append(vpcg, *g.VpcSecurityGroupId)
	}
	if err := d.Set("vpc_security_group_ids", vpcg); err != nil {
		return fmt.Errorf("[DEBUG] Error saving VPC Security Group IDs to state for RDS Cluster (%s): %s", d.Id(), err)
	}

	var cm []string
	for _, m := range dbc.DBClusterMembers {
		cm = append(cm, *m.DBInstanceIdentifier)
	}
	if err := d.Set("cluster_members", cm); err != nil {
		return fmt.Errorf("[DEBUG] Error saving RDS Cluster Members to state for RDS Cluster (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceAwsRDSClusterUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	req := &rds.ModifyDBClusterInput{
		ApplyImmediately:    aws.Bool(d.Get("apply_immediately").(bool)),
		DBClusterIdentifier: aws.String(d.Id()),
	}

	if d.HasChange("master_password") {
		req.MasterUserPassword = aws.String(d.Get("master_password").(string))
	}

	if d.HasChange("vpc_security_group_ids") {
		if attr := d.Get("vpc_security_group_ids").(*schema.Set); attr.Len() > 0 {
			req.VpcSecurityGroupIds = expandStringList(attr.List())
		} else {
			req.VpcSecurityGroupIds = []*string{}
		}
	}

	if d.HasChange("preferred_backup_window") {
		req.PreferredBackupWindow = aws.String(d.Get("preferred_backup_window").(string))
	}

	if d.HasChange("preferred_maintenance_window") {
		req.PreferredMaintenanceWindow = aws.String(d.Get("preferred_maintenance_window").(string))
	}

	if d.HasChange("backup_retention_period") {
		req.BackupRetentionPeriod = aws.Int64(int64(d.Get("backup_retention_period").(int)))
	}

	_, err := conn.ModifyDBCluster(req)
	if err != nil {
		return fmt.Errorf("[WARN] Error modifying RDS Cluster (%s): %s", d.Id(), err)
	}

	return resourceAwsRDSClusterRead(d, meta)
}

func resourceAwsRDSClusterDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn
	log.Printf("[DEBUG] Destroying RDS Cluster (%s)", d.Id())

	deleteOpts := rds.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String(d.Id()),
	}

	finalSnapshot := d.Get("final_snapshot_identifier").(string)
	if finalSnapshot == "" {
		deleteOpts.SkipFinalSnapshot = aws.Bool(true)
	} else {
		deleteOpts.FinalDBSnapshotIdentifier = aws.String(finalSnapshot)
		deleteOpts.SkipFinalSnapshot = aws.Bool(false)
	}

	log.Printf("[DEBUG] RDS Cluster delete options: %s", deleteOpts)
	_, err := conn.DeleteDBCluster(&deleteOpts)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"available", "deleting", "backing-up", "modifying"},
		Target:     []string{"destroyed"},
		Refresh:    resourceAwsRDSClusterStateRefreshFunc(d, meta),
		Timeout:    5 * time.Minute,
		MinTimeout: 3 * time.Second,
	}

	// Wait, catching any errors
	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("[WARN] Error deleting RDS Cluster (%s): %s", d.Id(), err)
	}

	return nil
}

func resourceAwsRDSClusterStateRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		conn := meta.(*AWSClient).rdsconn

		resp, err := conn.DescribeDBClusters(&rds.DescribeDBClustersInput{
			DBClusterIdentifier: aws.String(d.Id()),
		})

		if err != nil {
			if awsErr, ok := err.(awserr.Error); ok {
				if "DBClusterNotFoundFault" == awsErr.Code() {
					return 42, "destroyed", nil
				}
			}
			log.Printf("[WARN] Error on retrieving DB Cluster (%s) when waiting: %s", d.Id(), err)
			return nil, "", err
		}

		var dbc *rds.DBCluster

		for _, c := range resp.DBClusters {
			if *c.DBClusterIdentifier == d.Id() {
				dbc = c
			}
		}

		if dbc == nil {
			return 42, "destroyed", nil
		}

		if dbc.Status != nil {
			log.Printf("[DEBUG] DB Cluster status (%s): %s", d.Id(), *dbc.Status)
		}

		return dbc, *dbc.Status, nil
	}
}
