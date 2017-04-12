package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
)

func persistentVolumeSourceSchema() *schema.Resource {
	volumeSources["host_path"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "Represents a directory on the host. Provisioned by a developer or tester. This is useful for single-node development and testing only! On-host storage is not supported in any way and WILL NOT WORK in a multi-node cluster. More info: http://kubernetes.io/docs/user-guide/volumes#hostpath",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"path": {
					Type:        schema.TypeString,
					Description: "Path of the directory on the host. More info: http://kubernetes.io/docs/user-guide/volumes#hostpath",
					Optional:    true,
				},
			},
		},
	}
	return &schema.Resource{
		Schema: volumeSources,
	}
}

// Common volume sources between Persistent Volumes and Pod Volumes
var volumeSources = map[string]*schema.Schema{
	"aws_elastic_block_store": {
		Type:        schema.TypeList,
		Description: "Represents an AWS Disk resource that is attached to a kubelet's host machine and then exposed to the pod. More info: http://kubernetes.io/docs/user-guide/volumes#awselasticblockstore",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"fs_type": {
					Type:        schema.TypeString,
					Description: "Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. More info: http://kubernetes.io/docs/user-guide/volumes#awselasticblockstore",
					Optional:    true,
				},
				"partition": {
					Type:        schema.TypeInt,
					Description: "The partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as \"1\". Similarly, the volume partition for /dev/sda is \"0\" (or you can leave the property empty).",
					Optional:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to set the read-only property in VolumeMounts to \"true\". If omitted, the default is \"false\". More info: http://kubernetes.io/docs/user-guide/volumes#awselasticblockstore",
					Optional:    true,
				},
				"volume_id": {
					Type:        schema.TypeString,
					Description: "Unique ID of the persistent disk resource in AWS (Amazon EBS volume). More info: http://kubernetes.io/docs/user-guide/volumes#awselasticblockstore",
					Required:    true,
				},
			},
		},
	},
	"azure_disk": {
		Type:        schema.TypeList,
		Description: "Represents an Azure Data Disk mount on the host and bind mount to the pod.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"caching_mode": {
					Type:        schema.TypeString,
					Description: "Host Caching mode: None, Read Only, Read Write.",
					Required:    true,
				},
				"data_disk_uri": {
					Type:        schema.TypeString,
					Description: "The URI the data disk in the blob storage",
					Required:    true,
				},
				"disk_name": {
					Type:        schema.TypeString,
					Description: "The Name of the data disk in the blob storage",
					Required:    true,
				},
				"fs_type": {
					Type:        schema.TypeString,
					Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified.",
					Optional:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write).",
					Optional:    true,
					Default:     false,
				},
			},
		},
	},
	"azure_file": {
		Type:        schema.TypeList,
		Description: "Represents an Azure File Service mount on the host and bind mount to the pod.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write).",
					Optional:    true,
				},
				"secret_name": {
					Type:        schema.TypeString,
					Description: "The name of secret that contains Azure Storage Account Name and Key",
					Required:    true,
				},
				"share_name": {
					Type:        schema.TypeString,
					Description: "Share Name",
					Required:    true,
				},
			},
		},
	},
	"ceph_fs": {
		Type:        schema.TypeList,
		Description: "Represents a Ceph FS mount on the host that shares a pod's lifetime",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"monitors": {
					Type:        schema.TypeSet,
					Description: "Monitors is a collection of Ceph monitors More info: http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it",
					Required:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Set:         schema.HashString,
				},
				"path": {
					Type:        schema.TypeString,
					Description: "Used as the mounted root, rather than the full Ceph tree, default is /",
					Optional:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to force the read-only setting in VolumeMounts. Defaults to `false` (read/write). More info: http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it",
					Optional:    true,
				},
				"secret_file": {
					Type:        schema.TypeString,
					Description: "The path to key ring for User, default is /etc/ceph/user.secret More info: http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it",
					Optional:    true,
				},
				"secret_ref": {
					Type:        schema.TypeList,
					Description: "Reference to the authentication secret for User, default is empty. More info: http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it",
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:        schema.TypeString,
								Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
								Optional:    true,
							},
						},
					},
				},
				"user": {
					Type:        schema.TypeString,
					Description: "User is the rados user name, default is admin. More info: http://releases.k8s.io/HEAD/examples/volumes/cephfs/README.md#how-to-use-it",
					Optional:    true,
				},
			},
		},
	},
	"cinder": {
		Type:        schema.TypeList,
		Description: "Represents a cinder volume attached and mounted on kubelets host machine. More info: http://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"fs_type": {
					Type:        schema.TypeString,
					Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Examples: \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. More info: http://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md",
					Optional:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write). More info: http://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md",
					Optional:    true,
				},
				"volume_id": {
					Type:        schema.TypeString,
					Description: "Volume ID used to identify the volume in Cinder. More info: http://releases.k8s.io/HEAD/examples/mysql-cinder-pd/README.md",
					Required:    true,
				},
			},
		},
	},
	"fc": {
		Type:        schema.TypeList,
		Description: "Represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"fs_type": {
					Type:        schema.TypeString,
					Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified.",
					Optional:    true,
				},
				"lun": {
					Type:        schema.TypeInt,
					Description: "FC target lun number",
					Required:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to force the read-only setting in VolumeMounts. Defaults to false (read/write).",
					Optional:    true,
				},
				"target_ww_ns": {
					Type:        schema.TypeSet,
					Description: "FC target worldwide names (WWNs)",
					Required:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Set:         schema.HashString,
				},
			},
		},
	},
	"flex_volume": {
		Type:        schema.TypeList,
		Description: "Represents a generic volume resource that is provisioned/attached using an exec based plugin. This is an alpha feature and may change in future.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"driver": {
					Type:        schema.TypeString,
					Description: "Driver is the name of the driver to use for this volume.",
					Required:    true,
				},
				"fs_type": {
					Type:        schema.TypeString,
					Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". The default filesystem depends on FlexVolume script.",
					Optional:    true,
				},
				"options": {
					Type:        schema.TypeMap,
					Description: "Extra command options if any.",
					Optional:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to force the ReadOnly setting in VolumeMounts. Defaults to false (read/write).",
					Optional:    true,
				},
				"secret_ref": {
					Type:        schema.TypeList,
					Description: "Reference to the secret object containing sensitive information to pass to the plugin scripts. This may be empty if no secret object is specified. If the secret object contains more than one secret, all secrets are passed to the plugin scripts.",
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:        schema.TypeString,
								Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
								Optional:    true,
							},
						},
					},
				},
			},
		},
	},
	"flocker": {
		Type:        schema.TypeList,
		Description: "Represents a Flocker volume attached to a kubelet's host machine and exposed to the pod for its usage. This depends on the Flocker control service being running",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"dataset_name": {
					Type:        schema.TypeString,
					Description: "Name of the dataset stored as metadata -> name on the dataset for Flocker should be considered as deprecated",
					Optional:    true,
				},
				"dataset_uuid": {
					Type:        schema.TypeString,
					Description: "UUID of the dataset. This is unique identifier of a Flocker dataset",
					Optional:    true,
				},
			},
		},
	},
	"gce_persistent_disk": {
		Type:        schema.TypeList,
		Description: "Represents a GCE Disk resource that is attached to a kubelet's host machine and then exposed to the pod. Provisioned by an admin. More info: http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"fs_type": {
					Type:        schema.TypeString,
					Description: "Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. More info: http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk",
					Optional:    true,
				},
				"partition": {
					Type:        schema.TypeInt,
					Description: "The partition in the volume that you want to mount. If omitted, the default is to mount by volume name. Examples: For volume /dev/sda1, you specify the partition as \"1\". Similarly, the volume partition for /dev/sda is \"0\" (or you can leave the property empty). More info: http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk",
					Optional:    true,
				},
				"pd_name": {
					Type:        schema.TypeString,
					Description: "Unique name of the PD resource in GCE. Used to identify the disk in GCE. More info: http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk",
					Required:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to force the ReadOnly setting in VolumeMounts. Defaults to false. More info: http://kubernetes.io/docs/user-guide/volumes#gcepersistentdisk",
					Optional:    true,
				},
			},
		},
	},
	"glusterfs": {
		Type:        schema.TypeList,
		Description: "Represents a Glusterfs volume that is attached to a host and exposed to the pod. Provisioned by an admin. More info: http://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"endpoints_name": {
					Type:        schema.TypeString,
					Description: "The endpoint name that details Glusterfs topology. More info: http://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md#create-a-pod",
					Required:    true,
				},
				"path": {
					Type:        schema.TypeString,
					Description: "The Glusterfs volume path. More info: http://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md#create-a-pod",
					Required:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to force the Glusterfs volume to be mounted with read-only permissions. Defaults to false. More info: http://releases.k8s.io/HEAD/examples/volumes/glusterfs/README.md#create-a-pod",
					Optional:    true,
				},
			},
		},
	},
	"iscsi": {
		Type:        schema.TypeList,
		Description: "Represents an ISCSI Disk resource that is attached to a kubelet's host machine and then exposed to the pod. Provisioned by an admin.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"fs_type": {
					Type:        schema.TypeString,
					Description: "Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. More info: http://kubernetes.io/docs/user-guide/volumes#iscsi",
					Optional:    true,
				},
				"iqn": {
					Type:        schema.TypeString,
					Description: "Target iSCSI Qualified Name.",
					Required:    true,
				},
				"iscsi_interface": {
					Type:        schema.TypeString,
					Description: "iSCSI interface name that uses an iSCSI transport. Defaults to 'default' (tcp).",
					Optional:    true,
					Default:     "default",
				},
				"lun": {
					Type:        schema.TypeInt,
					Description: "iSCSI target lun number.",
					Optional:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to force the read-only setting in VolumeMounts. Defaults to false.",
					Optional:    true,
				},
				"target_portal": {
					Type:        schema.TypeString,
					Description: "iSCSI target portal. The portal is either an IP or ip_addr:port if the port is other than default (typically TCP ports 860 and 3260).",
					Required:    true,
				},
			},
		},
	},
	"nfs": {
		Type:        schema.TypeList,
		Description: "Represents an NFS mount on the host. Provisioned by an admin. More info: http://kubernetes.io/docs/user-guide/volumes#nfs",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"path": {
					Type:        schema.TypeString,
					Description: "Path that is exported by the NFS server. More info: http://kubernetes.io/docs/user-guide/volumes#nfs",
					Required:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to force the NFS export to be mounted with read-only permissions. Defaults to false. More info: http://kubernetes.io/docs/user-guide/volumes#nfs",
					Optional:    true,
				},
				"server": {
					Type:        schema.TypeString,
					Description: "Server is the hostname or IP address of the NFS server. More info: http://kubernetes.io/docs/user-guide/volumes#nfs",
					Required:    true,
				},
			},
		},
	},
	"photon_persistent_disk": {
		Type:        schema.TypeList,
		Description: "Represents a PhotonController persistent disk attached and mounted on kubelets host machine",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"fs_type": {
					Type:        schema.TypeString,
					Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified.",
					Optional:    true,
				},
				"pd_id": {
					Type:        schema.TypeString,
					Description: "ID that identifies Photon Controller persistent disk",
					Required:    true,
				},
			},
		},
	},
	"quobyte": {
		Type:        schema.TypeList,
		Description: "Quobyte represents a Quobyte mount on the host that shares a pod's lifetime",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"group": {
					Type:        schema.TypeString,
					Description: "Group to map volume access to Default is no group",
					Optional:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to force the Quobyte volume to be mounted with read-only permissions. Defaults to false.",
					Optional:    true,
				},
				"registry": {
					Type:        schema.TypeString,
					Description: "Registry represents a single or multiple Quobyte Registry services specified as a string as host:port pair (multiple entries are separated with commas) which acts as the central registry for volumes",
					Required:    true,
				},
				"user": {
					Type:        schema.TypeString,
					Description: "User to map volume access to Defaults to serivceaccount user",
					Optional:    true,
				},
				"volume": {
					Type:        schema.TypeString,
					Description: "Volume is a string that references an already created Quobyte volume by name.",
					Required:    true,
				},
			},
		},
	},
	"rbd": {
		Type:        schema.TypeList,
		Description: "Represents a Rados Block Device mount on the host that shares a pod's lifetime. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"ceph_monitors": {
					Type:        schema.TypeSet,
					Description: "A collection of Ceph monitors. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it",
					Required:    true,
					Elem:        &schema.Schema{Type: schema.TypeString},
					Set:         schema.HashString,
				},
				"fs_type": {
					Type:        schema.TypeString,
					Description: "Filesystem type of the volume that you want to mount. Tip: Ensure that the filesystem type is supported by the host operating system. Examples: \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified. More info: http://kubernetes.io/docs/user-guide/volumes#rbd",
					Optional:    true,
				},
				"keyring": {
					Type:        schema.TypeString,
					Description: "Keyring is the path to key ring for RBDUser. Default is /etc/ceph/keyring. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it",
					Optional:    true,
					Computed:    true,
				},
				"rados_user": {
					Type:        schema.TypeString,
					Description: "The rados user name. Default is admin. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it",
					Optional:    true,
					Default:     "admin",
				},
				"rbd_image": {
					Type:        schema.TypeString,
					Description: "The rados image name. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it",
					Required:    true,
				},
				"rbd_pool": {
					Type:        schema.TypeString,
					Description: "The rados pool name. Default is rbd. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it.",
					Optional:    true,
					Default:     "rbd",
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Whether to force the read-only setting in VolumeMounts. Defaults to false. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it",
					Optional:    true,
					Default:     false,
				},
				"secret_ref": {
					Type:        schema.TypeList,
					Description: "Name of the authentication secret for RBDUser. If provided overrides keyring. Default is nil. More info: http://releases.k8s.io/HEAD/examples/volumes/rbd/README.md#how-to-use-it",
					Optional:    true,
					MaxItems:    1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"name": {
								Type:        schema.TypeString,
								Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
								Optional:    true,
							},
						},
					},
				},
			},
		},
	},
	"vsphere_volume": {
		Type:        schema.TypeList,
		Description: "Represents a vSphere volume attached and mounted on kubelets host machine",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"fs_type": {
					Type:        schema.TypeString,
					Description: "Filesystem type to mount. Must be a filesystem type supported by the host operating system. Ex. \"ext4\", \"xfs\", \"ntfs\". Implicitly inferred to be \"ext4\" if unspecified.",
					Optional:    true,
				},
				"volume_path": {
					Type:        schema.TypeString,
					Description: "Path that identifies vSphere volume vmdk",
					Required:    true,
				},
			},
		},
	},
}
