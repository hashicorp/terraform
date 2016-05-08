---
layout: "aws"
page_title: "AWS: aws_redshift_cluster"
sidebar_current: "docs-aws-resource-redshift-cluster"
---

# aws\_redshift\_cluster

Provides a Redshift Cluster Resource. 

## Example Usage

```
resource "aws_redshift_cluster" "default" {
  cluster_identifier = "tf-redshift-cluster"
  database_name = "mydb"
  master_username = "foo"
  master_password = "Mustbe8characters"
  node_type = "dc1.large"
  cluster_type = "single-node"
}
```

## Argument Reference

For more detailed documentation about each argument, refer to
the [AWS official documentation](http://docs.aws.amazon.com/cli/latest/reference/redshift/index.html#cli-aws-redshift).

The following arguments are supported:

* `cluster_identifier` - (Required) The Cluster Identifier. Must be a lower case
string.
* `database_name` - (Optional) The name of the first database to be created when the cluster is created.
 If you do not provide a name, Amazon Redshift will create a default database called `dev`.
* `node_type` - (Required) The node type to be provisioned for the cluster.
* `master_password` - (Required) Password for the master DB user. Note that this may
    show up in logs, and it will be stored in the state file
* `master_username` - (Required) Username for the master DB user
* `cluster_security_groups` - (Optional) A list of security groups to be associated with this cluster.
* `vpc_security_group_ids` - (Optional) A list of Virtual Private Cloud (VPC) security groups to be associated with the cluster.
* `cluster_subnet_group_name` - (Optional) The name of a cluster subnet group to be associated with this cluster. If this parameter is not provided the resulting cluster will be deployed outside virtual private cloud (VPC).
* `availability_zone` - (Optional) The EC2 Availability Zone (AZ) in which you want Amazon Redshift to provision the cluster. For example, if you have several EC2 instances running in a specific Availability Zone, then you might want the cluster to be provisioned in the same zone in order to decrease network latency.
* `preferred_maintenance_window` - (Optional) The weekly time range (in UTC) during which automated cluster maintenance can occur.
                                              Format: ddd:hh24:mi-ddd:hh24:mi
* `cluster_parameter_group_name` - (Optional) The name of the parameter group to be associated with this cluster.
* `automated_snapshot_retention_period` - (Optional) The number of days that automated snapshots are retained. If the value is 0, automated snapshots are disabled. Even if automated snapshots are disabled, you can still create manual snapshots when you want with create-cluster-snapshot. Default is 1.
* `port` - (Optional) The port number on which the cluster accepts incoming connections.
                      The cluster is accessible only via the JDBC and ODBC connection strings. Part of the connection string requires the port on which the cluster will listen for incoming connections. Default port is 5439.
* `cluster_version` - (Optional) The version of the Amazon Redshift engine software that you want to deploy on the cluster.
                                 The version selected runs on all the nodes in the cluster.
* `allow_version_upgrade` - (Optional) If true , major version upgrades can be applied during the maintenance window to the Amazon Redshift engine that is running on the cluster. Default is true
* `number_of_nodes` - (Optional) The number of compute nodes in the cluster. This parameter is required when the ClusterType parameter is specified as multi-node. Default is 1.
* `publicly_accessible` - (Optional) If true, the cluster can be accessed from a public network. Default is `true`.
* `encrypted` - (Optional) If true , the data in the cluster is encrypted at rest.
* `elastic_ip` - (Optional) The Elastic IP (EIP) address for the cluster.
* `skip_final_snapshot` - (Optional) Determines whether a final snapshot of the cluster is created before Amazon Redshift deletes the cluster. If true , a final cluster snapshot is not created. If false , a final cluster snapshot is created before the cluster is deleted. Default is true.
* `final_snapshot_identifier` - (Optional) The identifier of the final snapshot that is to be created immediately before deleting the cluster. If this parameter is provided, `skip_final_snapshot` must be false.                                                                                                     

## Attributes Reference

The following attributes are exported:

* `id` - The Redshift Cluster ID.
* `cluster_identifier` - The Cluster Identifier
* `cluster_type` - The cluster type
* `node_type` - The type of nodes in the cluster
* `database_name` - The name of the default database in the Cluster
* `availability_zone` - The availability zone of the Cluster
* `automated_snapshot_retention_period` - The backup retention period
* `preferred_maintenance_window` - The backup window
* `endpoint` - The connection endpoint
* `encrypted` - Whether the data in the cluster is encrypted
* `cluster_security_groups` - The security groups associated with the cluster
* `vpc_security_group_ids` - The VPC security group Ids associated with the cluster
* `port` - The Port the cluster responds on
* `cluster_version` - The version of Redshift engine software
* `cluster_parameter_group_name` - The name of the parameter group to be associated with this cluster
* `cluster_subnet_group_name` - The name of a cluster subnet group to be associated with this cluster
* `cluster_public_key` - The public key for the cluster
* `cluster_revision_number` - The specific revision number of the database in the cluster 
	
