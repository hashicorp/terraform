package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
)

// Pod schema components

func genSecretRef() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func genFieldRef() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"api_version": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"field_path": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func genHostPathVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"path": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func genEmptyDirVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"medium": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func genGcePersistentDiskVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"pd_name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"fs_type": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"partition": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},

				"read_only": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func genAwsElasticBlockStoreVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"volume_id": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"fs_type": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"partition": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},

				"read_only": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func genGitRepoVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"repository": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"revision": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func genSecretVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"secret_name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func genNfsVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"server": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"path": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"read_only": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func genIscsiVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"target_portal": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"iqn": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"lun": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},

				"fs_type": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"read_only": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func genGlusterfsVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"endpoints_name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"path": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"read_only": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func genPersistentVolumeClaimVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"claim_name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"read_only": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func genRbdVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"ceph_monitors": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},

				"rbd_image": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"fs_type": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"rbd_pool": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"rados_user": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"keyring": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"secret_ref": genSecretRef(),

				"read_only": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func genCinderVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"volume_id": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"fs_type": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"read_only": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func genCephVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"monitors": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},

				"user": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"secret_file": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"secret_ref": genSecretRef(),

				"read_only": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func genFlockerVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"dataset_name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func genDownwardApiVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"items": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"path": &schema.Schema{
								Type:     schema.TypeString,
								Optional: true,
							},

							"field_ref": genFieldRef(),
						},
					},
				},
			},
		},
	}
}

func genFcVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"target_wwns": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},

				"lun": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},

				"fs_type": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"read_only": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func genVolumeSource() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true, // required
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"host_path": genHostPathVolumeSource(),

				"empty_dir": genEmptyDirVolumeSource(),

				"gce_persistent_disk": genGcePersistentDiskVolumeSource(),

				"aws_elastic_block_store": genAwsElasticBlockStoreVolumeSource(),

				"git_repo": genGitRepoVolumeSource(),

				"secret": genSecretVolumeSource(),

				"nfs": genNfsVolumeSource(),

				"iscsi": genIscsiVolumeSource(),

				"glusterfs": genGlusterfsVolumeSource(),

				"persistent_volume_claim": genPersistentVolumeClaimVolumeSource(),

				"rbd": genRbdVolumeSource(),

				"cinder": genCinderVolumeSource(),

				"ceph": genCephVolumeSource(),

				"flocker": genFlockerVolumeSource(),

				"downward_api": genDownwardApiVolumeSource(),

				"fc": genFcVolumeSource(),
			},
		},
	}
}

func genVolume() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true, // required
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true, // required
				},

				"volume_source": genVolumeSource(),
			},
		},
	}
}

func genContainerPort() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true, // required
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"host_port": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},

				"container_port": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true, // required
				},

				"protocol": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true, // required
				},

				"host_ip": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func genEnvVar() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true, // required
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true, // required
				},

				"value": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true, // required
				},

				"value_from": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"field_ref": genFieldRef(),
						},
					},
				},
			},
		},
	}
}

// There are two ways of doing this - either
// 1. an explicit struct mapping each `resourcename` to a quantity schema.
// 2. a map from `resourcename` (as a string) a quantity schema
// I'm going with 2. for now since it's more adaptive to changes in the
// kubernetes API
func genResourceList() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeMap,
		Optional: true,
		Elem:     genResourceQuantity(),
	}
}

func genResourceRequirements() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"limits": genResourceList(),

				"requests": genResourceList(),
			},
		},
	}
}

func genVolumeMount() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true, // required
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true, // required
				},

				"read_only": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},

				"mount_path": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true, // required
				},
			},
		},
	}
}

func genTcpSocketAction() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"port": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func genExecAction() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"command": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}

func genHttpGetAction() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"path": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"port": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"host": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"scheme": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func genHandler() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"exec": genExecAction(),

				"http_get": genHttpGetAction(),

				"tcp_socket": genTcpSocketAction(),
			},
		},
	}
}

func genLivenessProbe() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true, // required
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"handler": genHandler(),

				"initial_delay_seconds": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},

				"timeout_seconds": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},
			},
		},
	}
}

func genSeLinuxOptions() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"user": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"role": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"type": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"level": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}

func genCapabilities() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"add": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},

				"drop": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},
			},
		},
	}
}

func genSecurityContext() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"capabilities": genCapabilities(),

				"privileged": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},

				"se_linux_options": genSeLinuxOptions(),

				"run_as_user": &schema.Schema{
					Type:     schema.TypeInt,
					Optional: true,
				},

				"run_as_root": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func genContainer() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true, // required
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true, // required
				},

				"image": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true, // required
				},

				"command": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},

				"args": &schema.Schema{
					Type:     schema.TypeList,
					Optional: true,
					Elem:     &schema.Schema{Type: schema.TypeString},
				},

				"working_dir": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},

				"port": genContainerPort(),

				"env": genEnvVar(),

				"volume_mount": genVolumeMount(),

				"liveness_probe": genLivenessProbe(),

				"readiness_probe": genLivenessProbe(),

				"termination_message_path": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true, // required
				},

				"image_pull_path": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true, // required
				},

				"security_context": genSecurityContext(),

				"stdin": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},

				"tty": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func genPodSecurityContext() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true, // required
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"host_network": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},

				"host_pid": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},

				"host_ipc": &schema.Schema{
					Type:     schema.TypeBool,
					Optional: true,
				},
			},
		},
	}
}

func genLocalObjectReference() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"name": &schema.Schema{
					Type:     schema.TypeString,
					Optional: true,
				},
			},
		},
	}
}
