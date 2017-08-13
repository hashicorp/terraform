---
layout: "netapp"
page_title: "NetApp: netapp_cloud_volume"
sidebar_current: "docs-netapp-resource-cloud-volume"
description: |-
  Creates and manages NetApp OCCM volumes.
---

# netapp\_cloud\_volume

The ``netapp_cloud_volume`` resource creates and manages NetApp OCCM volumes.

## Example Usage

```hcl
resource "netapp_cloud_volume" "vsa-cifs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.vsa-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.vsa-workenv.svm_name}"
  name = "cifs_vol"
  type = "cifs"
  size = 1
  size_unit = "GB"
  snapshot_policy = "default"
  provider_volume_type = "sc1"
  share {
    name = "my_cifs_share"
    permission {
      type = "read"
      users = ["Everyone"]
    }
  }
}
```

```hcl
resource "netapp_cloud_volume" "vsa-nfs-volume" {
  workenv_id = "${data.netapp_cloud_workenv.vsa-workenv.public_id}"
  svm_name = "${data.netapp_cloud_workenv.vsa-workenv.svm_name}"
  name = "nfs_vol"
  type = "nfs"
  size = 1
  size_unit = "GB"
  snapshot_policy = "default"
  export_policy = ["10.11.12.13/32"]
  provider_volume_type = "gp2"
}
```

## Argument Reference

The following arguments are supported:

* `workenv_id` - (Required) Volume working environment ID.

* `svm_name` - (Required) Volume SVN name.

* `aggregate_name` - (Optional) Aggregate name where the volume should be created.

* `name` - (Required) Name for the volume.

* `type` - (Required) Volume type (`nfs` or `cifs`)

* `size` - (Required) Numeric volume size.

* `size_unit` - (Required) Size unit (`GB` or `TB`).

* `initial_size` - (Optional) Numeric initial volume size.

* `initial_size_unit` - (Optional) Initial size unit (`GB` or `TB`).

* `snapshot_policy` - (Optional) Snapshot policy name for volume.

* `export_policy` - (Optional) Export policy for NFS volumes (default value is `none`).

* `share` - (Required for CIFS) Share details for CIFS volumes.

* `thin_provisioning` - (Optional) Volume thin provisioning flag (default value is `true`).

* `compression` - (Optional) Volume compression flag (default value is `true`).

* `deduplication` - (Optional) Volume deduplication flag (default value is `true`).

* `max_num_disks_approved_to_add` - (Optional) Maximum number of disks allowed to be added when creating the volume.

* `verify_name_uniqueness` - (Optional) Indicates if name uniqueness check should be performed.

* `provider_volume_type` - (Optional) Disk type for the volume (`gp2`, `st1`, `io1` or `sc1`, default value is `gp2`).

* `iops` - (Required for `io1` disk type) IOPS to provision for the volume.

* `sync_to_s3` - (Optional) Indicates if volume data should be sent to S3. Only applicable for working environments that support S3 sync.

* `capacity_tier` - (Optional) Volume capacity tier (i.e. `S3`).

* `create_aggregate_if_not_found` - (Optional) Used with `aggregate_name`, if aggregate name does not exist, this flag controls whether the aggregate will get created.

## Import

Volumes can be imported using the `resource name`, e.g.

```shell
$ terraform import netapp_cloud_volume.vol1 example
```
