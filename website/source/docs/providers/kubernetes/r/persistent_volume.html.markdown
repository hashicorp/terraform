---
layout: "kubernetes"
page_title: "Kubernetes: kubernetes_persistent_volume"
sidebar_current: "docs-kubernetes-resource-persistent-volume-x"
description: |-
  A Persistent Volume (PV) is a piece of networked storage in the cluster that has been provisioned by an administrator.
---

# kubernetes_persistent_volume

The resource provides a piece of networked storage in the cluster provisioned by an administrator. It is a resource in the cluster just like a node is a cluster resource. Persistent Volumes have a lifecycle independent of any individual pod that uses the PV.

More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes/

## Example Usage

```hcl
resource "kubernetes_persistent_volume" "example" {
	metadata {
		name = "terraform-example"
	}
	spec {
		capacity {
			storage = "2Gi"
		}
		access_modes = ["ReadWriteMany"]
		persistent_volume_source {
			vsphere_volume {
				volume_path = "/absolute/path"
			}
		}
	}
}
```

## Argument Reference

The following arguments are supported:

* `metadata` - (Required) Standard persistent volume's metadata. More info: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
* `spec` - (Required) Spec of the persistent volume owned by the cluster. See below.

## Nested Blocks

### `spec`

#### Arguments

* `access_modes` - (Required) Contains all ways the volume can be mounted. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#access-modes
* `capacity` - (Required) A description of the persistent volume's resources and capacity. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#capacity
* `persistent_volume_reclaim_policy` - (Optional) What happens to a persistent volume when released from its claim. Valid options are Retain (default) and Recycle. Recycling must be supported by the volume plugin underlying this persistent volume. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#recycling-policy
* `persistent_volume_source` - (Required) The specification of a persistent volume.

### `persistent_volume_source`

#### Arguments

* `aws_elastic_block_store` - (Optional) Represents an AWS Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: http://kubernetes.io/docs/user-guide/volumes#awselasticblockstore
* `azure_disk` - (Optional) Represents an Azure Data Disk mount on the host and bind mount to the pod.
* `azure_file` - (Optional) Represents an Azure File Service mount on the host and bind mount to the pod.
* `ceph_fs` - (Optional) Represents a Ceph FS mount on the host that shares a pod's lifetime
* `cinder` - (Optional) Represents a cinder volume attached and mounted on kubelets host machine. More info: http://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md
* `fc` - (Optional) Represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod.
* `flex_volume` - (Optional) Represents a generic volume resource that is provisioned/attached using an exec based plugin. This is an alpha feature and may change in future.
* `flocker` - (Optional) Represents a Flocker volume attached to a kubelet's host machine and exposed to the pod for its usage. This depends on the Flocker control service being running
* `gce_persistent_disk` - (Optional) Represents a GCE Disk resource that is attached to a kubelet's host machine and then exposed to the pod. Provisioned by an admin. More info: http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk
* `glusterfs` - (Optional) Represents a Glusterfs volume that is attached to a host and exposed to the pod. Provisioned by an admin. More info: http://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md
* `host_path` - (Optional) Represents a directory on the host. Provisioned by a developer or tester. This is useful for single-node development and testing only! On-host storage is not supported in any way and WILL NOT WORK in a multi-node cluster. More info: http://kubernetes.io/docs/user-guide/volumes#hostpath
* `iscsi` - (Optional) Represents an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod. Provisioned by an admin.
* `nfs` - (Optional) Represents an NFS mount on the host. Provisioned by an admin. More info: http://kubernetes.io/docs/user-guide/volumes#nfs
* `photon_persistent_disk` - (Optional) Represents a PhotonController persistent disk attached and mounted on kubelets host machine
* `quobyte` - (Optional) Quobyte represents a Quobyte mount on the host that shares a pod's lifetime
* `rbd` - (Optional) Represents a Rados Block Device mount on the host that shares a pod's lifetime. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md
* `vsphere_volume` - (Optional) Represents a vSphere volume attached and mounted on kubelets host machine


### `aws_elastic_block_store`

#### Arguments

* `fs_type` - (Optional) Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: http://kubernetes.io/docs/user-guide/volumes#awselasticblockstore
* `partition` - (Optional) The partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1". Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty).
* `read_only` - (Optional) Whether to set the read-only property in VolumeMounts to "true". If omitted, the default is "false". More info: http://kubernetes.io/docs/user-guide/volumes#awselasticblockstore
* `volume_id` - (Required) Unique ID of the persistent disk resource in AWS (Amazon EBS volume). More info: http://kubernetes.io/docs/user-guide/volumes#awselasticblockstore

### `azure_disk`

#### Arguments

* `caching_mode` - (Required) Host Caching mode: None, Read Only, Read Write.
* `data_disk_uri` - (Required) The URI the data disk in the blob storage
* `disk_name` - (Required) The Name of the data disk in the blob storage
* `fs_type` - (Optional) Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.
* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write).

### `azure_file`

#### Arguments

* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write).
* `secret_name` - (Required) The name of secret that contains Azure Storage Account Name and Key
* `share_name` - (Required) Share Name

### `ceph_fs`

#### Arguments

* `monitors` - (Required) Monitors is a collection of Ceph monitors More info: http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it
* `path` - (Optional) Used as the mounted root, rather than the full Ceph tree, default is /
* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to `false` (read/write). More info: http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it
* `secret_file` - (Optional) The path to key ring for User, default is /etc/ceph/user.secret More info: http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it
* `secret_ref` - (Optional) Reference to the authentication secret for User, default is empty. More info: http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it
* `user` - (Optional) User is the rados user name, default is admin. More info: http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it

### `cinder`

#### Arguments

* `fs_type` - (Optional) Filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: http://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md
* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write). More info: http://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md
* `volume_id` - (Required) Volume ID used to identify the volume in Cinder. More info: http://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md

### `fc`

#### Arguments

* `fs_type` - (Optional) Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.
* `lun` - (Required) FC target lun number
* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write).
* `target_ww_ns` - (Required) FC target worldwide names (WWNs)

### `flex_volume`

#### Arguments

* `driver` - (Required) Driver is the name of the driver to use for this volume.
* `fs_type` - (Optional) Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". The default filesystem depends on FlexVolume script.
* `options` - (Optional) Extra command options if any.
* `read_only` - (Optional) Whether to force the ReadOnly setting in VolumeMounts. Defaults to false (read/write).
* `secret_ref` - (Optional) Reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified. If the secret object contains more than one secret, all secrets are passed to the plugin scripts.

### `flocker`

#### Arguments

* `dataset_name` - (Optional) Name of the dataset stored as metadata -> name on the dataset for Flocker should be considered as deprecated
* `dataset_uuid` - (Optional) UUID of the dataset. This is unique identifier of a Flocker dataset

### `gce_persistent_disk`

#### Arguments

* `fs_type` - (Optional) Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk
* `partition` - (Optional) The partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as "1". Similarly, the volume partition for /dev/sda is "0" (or you can leave the property empty). More info: http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk
* `pd_name` - (Required) Unique name of the PD resource in GCE. Used to identify the disk in GCE. More info: http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk
* `read_only` - (Optional) Whether to force the ReadOnly setting in VolumeMounts. Defaults to false. More info: http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk

### `glusterfs`

#### Arguments

* `endpoints_name` - (Required) The endpoint name that details Glusterfs topology. More info: http://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md#create-a-pod
* `path` - (Required) The Glusterfs volume path. More info: http://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md#create-a-pod
* `read_only` - (Optional) Whether to force the Glusterfs volume to be mounted with read-only permissions. Defaults to false. More info: http://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md#create-a-pod

### `host_path`

#### Arguments

* `path` - (Optional) Path of the directory on the host. More info: http://kubernetes.io/docs/user-guide/volumes#hostpath

### `iscsi`

#### Arguments

* `fs_type` - (Optional) Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: http://kubernetes.io/docs/user-guide/volumes#iscsi
* `iqn` - (Required) Target iSCSI Qualified Name.
* `iscsi_interface` - (Optional) iSCSI interface name that uses an iSCSI transport. Defaults to 'default' (tcp).
* `lun` - (Optional) iSCSI target lun number.
* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to false.
* `target_portal` - (Required) iSCSI target portal. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).

### `metadata`

#### Arguments

* `annotations` - (Optional) An unstructured key value map stored with the persistent volume that may be used to store arbitrary metadata. More info: http://kubernetes.io/docs/user-guide/annotations
* `labels` - (Optional) Map of string keys and values that can be used to organize and categorize (scope and select) the persistent volume. May match selectors of replication controllers and services. More info: http://kubernetes.io/docs/user-guide/labels
* `name` - (Optional) Name of the persistent volume, must be unique. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/identifiers#names

#### Attributes


* `generation` - A sequence number representing a specific generation of the desired state.
* `resource_version` - An opaque value that represents the internal version of this persistent volume that can be used by clients to determine when persistent volume has changed. Read more: https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#concurrency-control-and-consistency
* `self_link` - A URL representing this persistent volume.
* `uid` - The unique in time and space value for this persistent volume. More info: http://kubernetes.io/docs/user-guide/identifiers#uids

### `nfs`

#### Arguments

* `path` - (Required) Path that is exported by the NFS server. More info: http://kubernetes.io/docs/user-guide/volumes#nfs
* `read_only` - (Optional) Whether to force the NFS export to be mounted with read-only permissions. Defaults to false. More info: http://kubernetes.io/docs/user-guide/volumes#nfs
* `server` - (Required) Server is the hostname or IP address of the NFS server. More info: http://kubernetes.io/docs/user-guide/volumes#nfs

### `photon_persistent_disk`

#### Arguments

* `fs_type` - (Optional) Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.
* `pd_id` - (Required) ID that identifies Photon Controller persistent disk

### `quobyte`

#### Arguments

* `group` - (Optional) Group to map volume access to Default is no group
* `read_only` - (Optional) Whether to force the Quobyte volume to be mounted with read-only permissions. Defaults to false.
* `registry` - (Required) Registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes
* `user` - (Optional) User to map volume access to Defaults to serivceaccount user
* `volume` - (Required) Volume is a string that references an already created Quobyte volume by name.

### `rbd`

#### Arguments

* `ceph_monitors` - (Required) A collection of Ceph monitors. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it
* `fs_type` - (Optional) Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified. More info: http://kubernetes.io/docs/user-guide/volumes#rbd
* `keyring` - (Optional) Keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it
* `rados_user` - (Optional) The rados user name. Default is admin. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it
* `rbd_image` - (Required) The rados image name. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it
* `rbd_pool` - (Optional) The rados pool name. Default is rbd. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it.
* `read_only` - (Optional) Whether to force the read-only setting in VolumeMounts. Defaults to false. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it
* `secret_ref` - (Optional) Name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it

### `secret_ref`

#### Arguments

* `name` - (Optional) Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names

### `vsphere_volume`

#### Arguments

* `fs_type` - (Optional) Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. "ext4", "xfs", "ntfs". Implicitly inferred to be "ext4" if unspecified.
* `volume_path` - (Required) Path that identifies vSphere volume vmdk

## Import

Persistent Volume can be imported using its name, e.g.

```
$ terraform import kubernetes_persistent_volume.example terraform-example
```
