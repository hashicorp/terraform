---
layout: "aws"
page_title: "AWS: aws_db_snapshot"
sidebar_current: "docs-aws-resource-db-snapshot"
description: |-
  Provides an DB Instance.
---

# aws\_db\_snapshot

Creates a Snapshot of an DB Instance.

## Example Usage

```
resource "aws_db_instance" "bar" {
	allocated_storage = 10
	engine = "MySQL"
	engine_version = "5.6.21"
	instance_class = "db.t1.micro"
	name = "baz"
	password = "barbarbarbar"
	username = "foo"

    maintenance_window = "Fri:09:00-Fri:09:30"
	backup_retention_period = 0
	parameter_group_name = "default.mysql5.6"
}

resource "aws_db_snapshot" "test" {
	db_instance_identifier = "${aws_db_instance.bar.id}"
	db_snapshot_identifier = "testsnapshot1234"
}
```

## Argument Reference

The following arguments are supported:

* `db_instance_identifier` - (Required) The DB Instance Identifier from which to take the snapshot.
* `db_snapshot_identifier` - (Required) The Identifier for the snapshot.


## Attributes Reference

The following attributes are exported:

* `allocated_storage` - Specifies the allocated storage size in gigabytes (GB).
* `availability_zone` - Specifies the name of the Availability Zone the DB instance was located in at the time of the DB snapshot.
* `db_snapshot_arn` - The Amazon Resource Name (ARN) for the DB snapshot.
* `encrypted` - Specifies whether the DB snapshot is encrypted.
* `engine` - Specifies the name of the database engine.
* `engine_version` - Specifies the version of the database engine.
* `iops` - Specifies the Provisioned IOPS (I/O operations per second) value of the DB instance at the time of the snapshot.
* `kms_key_id` - The ARN for the KMS encryption key.
* `license_model` - License model information for the restored DB instance.
* `option_group_name` - Provides the option group name for the DB snapshot.
* `source_db_snapshot_identifier` - The DB snapshot Arn that the DB snapshot was copied from. It only has value in case of cross customer or cross region copy.
* `source_region` - The region that the DB snapshot was created in or copied from.
* `status` - Specifies the status of this DB snapshot.
* `storage_type` - Specifies the storage type associated with DB snapshot.
* `vpc_id` - Specifies the storage type associated with DB snapshot.