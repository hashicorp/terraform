package aws

import (
	"log"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsRdsCluster() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsRdsClusterRead,
		Schema: map[string]*schema.Schema{
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

	params := &rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(d.Get("cluster_identifier").(string)),
	}
	log.Printf("[DEBUG] Reading RDS Cluster: %s", params)
	resp, err := conn.DescribeDBClusters(params)

	if err != nil {
		return errwrap.Wrapf("Error retrieving RDS cluster: {{err}}", err)
	}

	d.SetId(*resp.DBClusters[0].DBClusterIdentifier)

	return flattenAwsRdsClusterResource(d, meta, resp.DBClusters[0])
}
