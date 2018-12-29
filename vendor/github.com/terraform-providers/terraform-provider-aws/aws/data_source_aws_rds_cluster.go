package aws

import (
	"fmt"
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsRdsCluster() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsRdsClusterRead,
		Schema: map[string]*schema.Schema{
			"arn": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cluster_identifier": {
				Type:     schema.TypeString,
				Required: true,
			},

			"availability_zones": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},

			"backup_retention_period": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"cluster_members": {
				Type:     schema.TypeSet,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Computed: true,
				Set:      schema.HashString,
			},

			"cluster_resource_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"database_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"db_subnet_group_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"db_cluster_parameter_group_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"enabled_cloudwatch_logs_exports": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"endpoint": {
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

			"final_snapshot_identifier": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"iam_database_authentication_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"iam_roles": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"kms_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"master_username": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"preferred_backup_window": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"preferred_maintenance_window": {
				Type:     schema.TypeString,
				Computed: true,
				StateFunc: func(val interface{}) string {
					if val == nil {
						return ""
					}
					return strings.ToLower(val.(string))
				},
			},

			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"reader_endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"replication_source_identifier": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"storage_encrypted": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"tags": tagsSchemaComputed(),

			"vpc_security_group_ids": {
				Type:     schema.TypeSet,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func dataSourceAwsRdsClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).rdsconn

	dbClusterIdentifier := d.Get("cluster_identifier").(string)

	params := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(dbClusterIdentifier),
	}
	log.Printf("[DEBUG] Reading RDS Cluster: %s", params)
	resp, err := conn.DescribeDBClusters(params)

	if err != nil {
		return fmt.Errorf("Error retrieving RDS cluster: %s", err)
	}

	if resp == nil {
		return fmt.Errorf("Error retrieving RDS cluster: empty response for: %s", params)
	}

	var dbc *rds.DBCluster
	for _, c := range resp.DBClusters {
		if aws.StringValue(c.DBClusterIdentifier) == dbClusterIdentifier {
			dbc = c
			break
		}
	}

	if dbc == nil {
		return fmt.Errorf("Error retrieving RDS cluster: cluster not found in response for: %s", params)
	}

	d.SetId(aws.StringValue(dbc.DBClusterIdentifier))

	if err := d.Set("availability_zones", aws.StringValueSlice(dbc.AvailabilityZones)); err != nil {
		return fmt.Errorf("error setting availability_zones: %s", err)
	}

	d.Set("arn", dbc.DBClusterArn)
	d.Set("backtrack_window", int(aws.Int64Value(dbc.BacktrackWindow)))
	d.Set("backup_retention_period", dbc.BackupRetentionPeriod)
	d.Set("cluster_identifier", dbc.DBClusterIdentifier)

	var cm []string
	for _, m := range dbc.DBClusterMembers {
		cm = append(cm, aws.StringValue(m.DBInstanceIdentifier))
	}
	if err := d.Set("cluster_members", cm); err != nil {
		return fmt.Errorf("error setting cluster_members: %s", err)
	}

	d.Set("cluster_resource_id", dbc.DbClusterResourceId)

	// Only set the DatabaseName if it is not nil. There is a known API bug where
	// RDS accepts a DatabaseName but does not return it, causing a perpetual
	// diff.
	//	See https://github.com/hashicorp/terraform/issues/4671 for backstory
	if dbc.DatabaseName != nil {
		d.Set("database_name", dbc.DatabaseName)
	}

	d.Set("db_cluster_parameter_group_name", dbc.DBClusterParameterGroup)
	d.Set("db_subnet_group_name", dbc.DBSubnetGroup)

	if err := d.Set("enabled_cloudwatch_logs_exports", aws.StringValueSlice(dbc.EnabledCloudwatchLogsExports)); err != nil {
		return fmt.Errorf("error setting enabled_cloudwatch_logs_exports: %s", err)
	}

	d.Set("endpoint", dbc.Endpoint)
	d.Set("engine_version", dbc.EngineVersion)
	d.Set("engine", dbc.Engine)
	d.Set("hosted_zone_id", dbc.HostedZoneId)
	d.Set("iam_database_authentication_enabled", dbc.IAMDatabaseAuthenticationEnabled)

	var roles []string
	for _, r := range dbc.AssociatedRoles {
		roles = append(roles, aws.StringValue(r.RoleArn))
	}
	if err := d.Set("iam_roles", roles); err != nil {
		return fmt.Errorf("error setting iam_roles: %s", err)
	}

	d.Set("kms_key_id", dbc.KmsKeyId)
	d.Set("master_username", dbc.MasterUsername)
	d.Set("port", dbc.Port)
	d.Set("preferred_backup_window", dbc.PreferredBackupWindow)
	d.Set("preferred_maintenance_window", dbc.PreferredMaintenanceWindow)
	d.Set("reader_endpoint", dbc.ReaderEndpoint)
	d.Set("replication_source_identifier", dbc.ReplicationSourceIdentifier)

	d.Set("storage_encrypted", dbc.StorageEncrypted)

	var vpcg []string
	for _, g := range dbc.VpcSecurityGroups {
		vpcg = append(vpcg, aws.StringValue(g.VpcSecurityGroupId))
	}
	if err := d.Set("vpc_security_group_ids", vpcg); err != nil {
		return fmt.Errorf("error setting vpc_security_group_ids: %s", err)
	}

	// Fetch and save tags
	if err := saveTagsRDS(conn, d, aws.StringValue(dbc.DBClusterArn)); err != nil {
		log.Printf("[WARN] Failed to save tags for RDS Cluster (%s): %s", aws.StringValue(dbc.DBClusterIdentifier), err)
	}

	return nil
}
