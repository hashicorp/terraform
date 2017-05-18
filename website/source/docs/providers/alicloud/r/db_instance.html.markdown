---
layout: "alicloud"
page_title: "Alicloud: alicloud_db_instance"
sidebar_current: "docs-alicloud-resource-db-instance"
description: |-
  Provides an RDS instance resource.
---

# alicloud\_db\_instance

Provides an RDS instance resource.  A DB instance is an isolated database
environment in the cloud.  A DB instance can contain multiple user-created
databases.

## Example Usage

```
resource "alicloud_db_instance" "default" {
	engine = "MySQL"
	engine_version = "5.6"
	db_instance_class = "rds.mysql.t1.small"
	db_instance_storage = "10"
	db_instance_net_type = "Intranet"
}
```

## Argument Reference

The following arguments are supported:

* `engine` - (Required) Database type. Value options: MySQL, SQLServer, PostgreSQL, and PPAS.
* `engine_version` - (Required) Database version. Value options: 
    - 5.5/5.6/5.7 for MySQL
    - 2008r2/2012 for SQLServer
    - 9.4 for PostgreSQL
    - 9.3 for PPAS
* `db_instance_class` - (Required) Instance type. For details, see [Instance type table](https://intl.aliyun.com/help/doc-detail/26312.htm?spm=a3c0i.o26228en.a3.2.bRUHF3).
* `db_instance_storage` - (Required) User-defined storage space. Value range: 
    - [5, 2000] for MySQL/PostgreSQL/PPAS HA dual node edition;
    - [20,1000] for MySQL 5.7 basic single node edition;
    - [10, 2000] for SQL Server 2008R2;
    - [20,2000] for SQL Server 2012 basic single node edition
    Increase progressively at a rate of 5 GB. The unit is GB. For details, see [Instance type table](https://intl.aliyun.com/help/doc-detail/26312.htm?spm=a3c0i.o26228en.a3.3.bRUHF3).
* `instance_charge_type` - (Optional) Valid values are `Prepaid`, `Postpaid`, The default is `Postpaid`.
* `period` - (Optional) The time that you have bought the resource, in month. Only valid when instance_charge_type is set as `PrePaid`. Value range [1, 12].
* `zone_id` - (Optional) Selected zone to create database instance. You cannot set the ZoneId parameter if the MultiAZ parameter is set to true.
* `multi_az` - (Optional) Specifies if the database instance is a multiple Availability Zone deployment.
* `db_instance_net_type` - (Optional) Network connection type of an instance. Internet: public network; Intranet: private network
* `allocate_public_connection` - (Optional) If set to true will applies for an Internet connection string of an instance.
* `instance_network_type` - (Optional) VPC: VPC instance; Classic: classic instance. If no value is specified, a classic instance will be created by default.
* `vswitch_id` - (Optional) The virtual switch ID to launch in VPC. If you want to create instances in VPC network, this parameter must be set.
* `master_user_name` - (Optional) The master user name for the database instance. Operation account requiring a uniqueness check. It may consist of lower case letters, numbers and underlines, and must start with a letter and have no more than 16 characters.
* `master_user_password` - (Optional) The master password for the database instance. Operation password. It may consist of letters, digits, or underlines, with a length of 6 to 32 characters.
* `preferred_backup_period` - (Optional) Backup period. Values: Monday, Tuesday, Wednesday, Thursday, Friday, Saturday, and Sunday.
* `preferred_backup_time` - (Optional) Backup time, in the format ofHH:mmZ- HH:mm Z.
* `backup_retention_period` - (Optional) Retention days of the backup (7 to 730 days). The default value is 7 days.
* `security_ips` - (Optional) List of IP addresses under the IP address white list array. The list contains up to 1,000 IP addresses, separated by commas. Supported formats include 0.0.0.0/0, 10.23.12.24 (IP), and 10.23.12.24/24 (Classless Inter-Domain Routing (CIDR) mode. /24 represents the length of the prefix in an IP address. The range of the prefix length is [1,32]).
* `db_mappings` - (Optional) Database mappings to attach to db instance. See [Block database](#block-database) below for details.


## Block database

The database mapping supports the following:

* `db_name` - (Required) Name of the database requiring a uniqueness check. It may consist of lower case letters, numbers and underlines, and must start with a letter and have no more than 64 characters. 
* `character_set_name` - (Required) Character set. The value range is limited to the following:
    - MySQL type:
         + utf8
         + gbk
         + latin1
         + utf8mb4 (included in versions 5.5 and 5.6).
    - SQLServer type:
         + Chinese_PRC_CI_AS
         + Chinese_PRC_CS_AS
         + SQL_Latin1_General_CP1_CI_AS
         + SQL_Latin1_General_CP1_CS_AS
         + Chinese_PRC_BIN
* `db_description` - (Optional) Database description, which cannot exceed 256 characters. NOTE: It cannot begin with https://. 


~> **NOTE:** We neither support modify any of database attribute, nor insert/remove item at the same time.
We recommend split to two separate operations.

## Attributes Reference

The following attributes are exported:

* `id` - The RDS instance ID.
* `instance_charge_type` - The instance charge type.
* `period` - The time that you have bought the resource.
* `engine` - Database type.
* `engine_version` - The database engine version.
* `db_instance_class` - The RDS instance class.
* `db_instance_storage` - The amount of allocated storage.
* `port` - The database port.
* `zone_id` - The zone ID of the DB instance.
* `db_instance_net_type` - Network connection type of an instance, `Internet` or `Intranet`.
* `instance_network_type` - The instance network type and it has two values: `vpc` and `classic`.
* `db_mappings` - Database mappings attached to db instance.
* `preferred_backup_period` - Backup period.
* `preferred_backup_time` - Backup time.
* `backup_retention_period` - Retention days of the backup.
* `security_ips` - Security ips of instance whitelist.
* `connections` - Views all the connection information of a specified instance.

