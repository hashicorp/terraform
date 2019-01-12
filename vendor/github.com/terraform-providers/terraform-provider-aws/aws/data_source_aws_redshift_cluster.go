package aws

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/redshift"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceAwsRedshiftCluster() *schema.Resource {
	return &schema.Resource{
		Read: dataSourceAwsRedshiftClusterRead,

		Schema: map[string]*schema.Schema{

			"cluster_identifier": {
				Type:     schema.TypeString,
				Required: true,
			},

			"allow_version_upgrade": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"automated_snapshot_retention_period": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"availability_zone": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"bucket_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cluster_parameter_group_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cluster_public_key": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cluster_revision_number": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cluster_security_groups": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"cluster_subnet_group_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cluster_type": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"cluster_version": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"database_name": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"elastic_ip": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"enable_logging": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"encrypted": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"endpoint": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"enhanced_vpc_routing": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"iam_roles": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"kms_key_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"master_username": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"node_type": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"number_of_nodes": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"port": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"preferred_maintenance_window": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"publicly_accessible": {
				Type:     schema.TypeBool,
				Computed: true,
			},

			"s3_key_prefix": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"tags": tagsSchema(),

			"vpc_id": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"vpc_security_group_ids": {
				Type:     schema.TypeList,
				Computed: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
		},
	}
}

func dataSourceAwsRedshiftClusterRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).redshiftconn

	cluster := d.Get("cluster_identifier").(string)

	log.Printf("[INFO] Reading Redshift Cluster Information: %s", cluster)

	resp, err := conn.DescribeClusters(&redshift.DescribeClustersInput{
		ClusterIdentifier: aws.String(cluster),
	})

	if err != nil {
		return fmt.Errorf("Error describing Redshift Cluster: %s, error: %s", cluster, err)
	}

	if resp.Clusters == nil || len(resp.Clusters) == 0 {
		return fmt.Errorf("Error describing Redshift Cluster: %s, cluster information not found", cluster)
	}

	rsc := *resp.Clusters[0]

	d.SetId(cluster)
	d.Set("allow_version_upgrade", rsc.AllowVersionUpgrade)
	d.Set("automated_snapshot_retention_period", rsc.AutomatedSnapshotRetentionPeriod)
	d.Set("availability_zone", rsc.AvailabilityZone)
	d.Set("cluster_identifier", rsc.ClusterIdentifier)

	if len(rsc.ClusterParameterGroups) > 0 {
		d.Set("cluster_parameter_group_name", rsc.ClusterParameterGroups[0].ParameterGroupName)
	}

	d.Set("cluster_public_key", rsc.ClusterPublicKey)
	d.Set("cluster_revision_number", rsc.ClusterRevisionNumber)

	var csg []string
	for _, g := range rsc.ClusterSecurityGroups {
		csg = append(csg, *g.ClusterSecurityGroupName)
	}
	if err := d.Set("cluster_security_groups", csg); err != nil {
		return fmt.Errorf("Error saving Cluster Security Group Names to state for Redshift Cluster (%s): %s", cluster, err)
	}

	d.Set("cluster_subnet_group_name", rsc.ClusterSubnetGroupName)

	if len(rsc.ClusterNodes) > 1 {
		d.Set("cluster_type", "multi-node")
	} else {
		d.Set("cluster_type", "single-node")
	}

	d.Set("cluster_version", rsc.ClusterVersion)
	d.Set("database_name", rsc.DBName)

	if rsc.ElasticIpStatus != nil {
		d.Set("elastic_ip", rsc.ElasticIpStatus.ElasticIp)
	}

	d.Set("encrypted", rsc.Encrypted)

	if rsc.Endpoint != nil {
		d.Set("endpoint", rsc.Endpoint.Address)
	}

	d.Set("enhanced_vpc_routing", rsc.EnhancedVpcRouting)

	var iamRoles []string
	for _, i := range rsc.IamRoles {
		iamRoles = append(iamRoles, *i.IamRoleArn)
	}
	if err := d.Set("iam_roles", iamRoles); err != nil {
		return fmt.Errorf("Error saving IAM Roles to state for Redshift Cluster (%s): %s", cluster, err)
	}

	d.Set("kms_key_id", rsc.KmsKeyId)
	d.Set("master_username", rsc.MasterUsername)
	d.Set("node_type", rsc.NodeType)
	d.Set("number_of_nodes", rsc.NumberOfNodes)
	d.Set("port", rsc.Endpoint.Port)
	d.Set("preferred_maintenance_window", rsc.PreferredMaintenanceWindow)
	d.Set("publicly_accessible", rsc.PubliclyAccessible)
	d.Set("tags", tagsToMapRedshift(rsc.Tags))
	d.Set("vpc_id", rsc.VpcId)

	var vpcg []string
	for _, g := range rsc.VpcSecurityGroups {
		vpcg = append(vpcg, *g.VpcSecurityGroupId)
	}
	if err := d.Set("vpc_security_group_ids", vpcg); err != nil {
		return fmt.Errorf("Error saving VPC Security Group IDs to state for Redshift Cluster (%s): %s", cluster, err)
	}

	log.Printf("[INFO] Reading Redshift Cluster Logging Status: %s", cluster)
	loggingStatus, loggingErr := conn.DescribeLoggingStatus(&redshift.DescribeLoggingStatusInput{
		ClusterIdentifier: aws.String(cluster),
	})

	if loggingErr != nil {
		return loggingErr
	}

	if loggingStatus != nil && aws.BoolValue(loggingStatus.LoggingEnabled) {
		d.Set("enable_logging", loggingStatus.LoggingEnabled)
		d.Set("bucket_name", loggingStatus.BucketName)
		d.Set("s3_key_prefix", loggingStatus.S3KeyPrefix)
	}

	return nil
}
