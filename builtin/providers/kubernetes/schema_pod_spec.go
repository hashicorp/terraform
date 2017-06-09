package kubernetes

import "github.com/hashicorp/terraform/helper/schema"

func podSpecFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"active_deadline_seconds": {
			Type:         schema.TypeInt,
			Optional:     true,
			ValidateFunc: validatePositiveInteger,
			Description:  "Optional duration in seconds the pod may be active on the node relative to StartTime before the system will actively try to mark it failed and kill associated containers. Value must be a positive integer.",
		},
		"container": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "List of containers belonging to the pod. Containers cannot currently be added or removed. There must be at least one container in a Pod. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/containers",
			Elem: &schema.Resource{
				Schema: containerFields(),
			},
		},
		"dns_policy": {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "ClusterFirst",
			Description: "Set DNS policy for containers within the pod. One of 'ClusterFirst' or 'Default'. Defaults to 'ClusterFirst'.",
		},
		"host_ipc": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Use the host's ipc namespace. Optional: Default to false.",
		},
		"host_network": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Host networking requested for this pod. Use the host's network namespace. If this option is set, the ports that will be used must be specified.",
		},

		"host_pid": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Use the host's pid namespace.",
		},

		"hostname": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Specifies the hostname of the Pod If not specified, the pod's hostname will be set to a system-defined value.",
		},
		"image_pull_secrets": {
			Type:        schema.TypeList,
			Description: "ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec. If specified, these secrets will be passed to individual puller implementations for them to use. For example, in the case of docker, only DockerConfig type secrets are honored. More info: http://kubernetes.io/docs/user-guide/images#specifying-imagepullsecrets-on-a-pod",
			Optional:    true,
			Computed:    true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"name": {
						Type:        schema.TypeString,
						Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
						Required:    true,
					},
				},
			},
		},
		"node_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "NodeName is a request to schedule this pod onto a specific node. If it is non-empty, the scheduler simply schedules this pod onto that node, assuming that it fits resource requirements.",
		},
		"node_selector": {
			Type:        schema.TypeMap,
			Optional:    true,
			Description: "NodeSelector is a selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node. More info: http://kubernetes.io/docs/user-guide/node-selection.",
		},
		"restart_policy": {
			Type:        schema.TypeString,
			Optional:    true,
			Default:     "Always",
			Description: "Restart policy for all containers within the pod. One of Always, OnFailure, Never. More info: http://kubernetes.io/docs/user-guide/pod-states#restartpolicy.",
		},
		"security_context": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "SecurityContext holds pod-level security attributes and common container settings. Optional: Defaults to empty",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"fs_group": {
						Type:        schema.TypeInt,
						Description: "A special supplemental group that applies to all containers in a pod. Some volume types allow the Kubelet to change the ownership of that volume to be owned by the pod: 1. The owning GID will be the FSGroup 2. The setgid bit is set (new files created in the volume will be owned by FSGroup) 3. The permission bits are OR'd with rw-rw---- If unset, the Kubelet will not modify the ownership and permissions of any volume.",
						Optional:    true,
					},
					"run_as_non_root": {
						Type:        schema.TypeBool,
						Description: "Indicates that the container must run as a non-root user. If true, the Kubelet will validate the image at runtime to ensure that it does not run as UID 0 (root) and fail to start the container if it does.",
						Optional:    true,
					},
					"run_as_user": {
						Type:        schema.TypeInt,
						Description: "The UID to run the entrypoint of the container process. Defaults to user specified in image metadata if unspecified",
						Optional:    true,
					},
					"supplemental_groups": {
						Type:        schema.TypeSet,
						Description: "A list of groups applied to the first process run in each container, in addition to the container's primary GID. If unspecified, no groups will be added to any container.",
						Optional:    true,
						Elem: &schema.Schema{
							Type: schema.TypeInt,
						},
					},
					"se_linux_options": {
						Type:        schema.TypeList,
						Description: "The SELinux context to be applied to all containers. If unspecified, the container runtime will allocate a random SELinux context for each container. May also be set in SecurityContext. If set in both SecurityContext and PodSecurityContext, the value specified in SecurityContext takes precedence for that container.",
						Optional:    true,
						MaxItems:    1,
						Elem: &schema.Resource{
							Schema: seLinuxOptionsField(),
						},
					},
				},
			},
		},
		"service_account_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "ServiceAccountName is the name of the ServiceAccount to use to run this pod. More info: http://releases.k8s.io/HEAD/docs/design/service_accounts.md.",
		},
		"subdomain": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: `If specified, the fully qualified Pod hostname will be "...svc.". If not specified, the pod will not have a domainname at all..`,
		},
		"termination_grace_period_seconds": {
			Type:         schema.TypeInt,
			Optional:     true,
			Default:      30,
			ValidateFunc: validateTerminationGracePeriodSeconds,
			Description:  "Optional duration in seconds the pod needs to terminate gracefully. May be decreased in delete request. Value must be non-negative integer. The value zero indicates delete immediately. If this value is nil, the default grace period will be used instead. The grace period is the duration in seconds after the processes running in the pod are sent a termination signal and the time when the processes are forcibly halted with a kill signal. Set this value longer than the expected cleanup time for your process.",
		},

		"volume": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "List of volumes that can be mounted by containers belonging to the pod. More info: http://kubernetes.io/docs/user-guide/volumes",
			Elem:        volumeSchema(),
		},
	}
}

func volumeSchema() *schema.Resource {
	v := commonVolumeSources()

	v["config_map"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "ConfigMap represents a configMap that should populate this volume",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"items": {
					Type:        schema.TypeList,
					Description: `If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error. Paths must be relative and may not contain the '..' path or start with '..'.`,
					Optional:    true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"key": {
								Type:        schema.TypeString,
								Optional:    true,
								Description: "The key to project.",
							},
							"mode": {
								Type:        schema.TypeInt,
								Optional:    true,
								Description: `Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
							},
							"path": {
								Type:         schema.TypeString,
								Optional:     true,
								ValidateFunc: validateAttributeValueDoesNotContain(".."),
								Description:  `The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.`,
							},
						},
					},
				},
				"default_mode": {
					Type:        schema.TypeInt,
					Description: "Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
					Optional:    true,
				},
				"name": {
					Type:        schema.TypeString,
					Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
					Optional:    true,
				},
			},
		},
	}

	v["git_repo"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "GitRepo represents a git repository at a particular revision.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"directory": {
					Type:         schema.TypeString,
					Description:  "Target directory name. Must not contain or start with '..'. If '.' is supplied, the volume directory will be the git repository. Otherwise, if specified, the volume will contain the git repository in the subdirectory with the given name.",
					Optional:     true,
					ValidateFunc: validateAttributeValueDoesNotContain(".."),
				},
				"repository": {
					Type:        schema.TypeString,
					Description: "Repository URL",
					Optional:    true,
				},
				"revision": {
					Type:        schema.TypeString,
					Description: "Commit hash for the specified revision.",
					Optional:    true,
				},
			},
		},
	}
	v["downward_api"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "DownwardAPI represents downward API about the pod that should populate this volume",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"default_mode": {
					Type:        schema.TypeInt,
					Description: "Optional: mode bits to use on created files by default. Must be a value between 0 and 0777. Defaults to 0644. Directories within the path are not affected by this setting. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.",
					Optional:    true,
				},
				"items": {
					Type:        schema.TypeList,
					Description: `If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error. Paths must be relative and may not contain the '..' path or start with '..'.`,
					Optional:    true,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"field_ref": {
								Type:        schema.TypeList,
								Required:    true,
								MaxItems:    1,
								Description: "Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.",
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"api_version": {
											Type:        schema.TypeString,
											Optional:    true,
											Default:     "v1",
											Description: `Version of the schema the FieldPath is written in terms of, defaults to "v1".`,
										},
										"field_path": {
											Type:        schema.TypeString,
											Optional:    true,
											Description: "Path of the field to select in the specified API version",
										},
									},
								},
							},
							"mode": {
								Type:        schema.TypeInt,
								Optional:    true,
								Description: `Optional: mode bits to use on this file, must be a value between 0 and 0777. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.`,
							},
							"path": {
								Type:         schema.TypeString,
								Required:     true,
								ValidateFunc: validateAttributeValueDoesNotContain(".."),
								Description:  `Path is the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'`,
							},
							"resource_field_ref": {
								Type:        schema.TypeList,
								Optional:    true,
								MaxItems:    1,
								Description: "Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.",
								Elem: &schema.Resource{
									Schema: map[string]*schema.Schema{
										"container_name": {
											Type:     schema.TypeString,
											Required: true,
										},
										"quantity": {
											Type:     schema.TypeString,
											Optional: true,
										},
										"resource": {
											Type:        schema.TypeString,
											Required:    true,
											Description: "Resource to select",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	v["empty_dir"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "EmptyDir represents a temporary directory that shares a pod's lifetime. More info: http://kubernetes.io/docs/user-guide/volumes#emptydir",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"medium": {
					Type:         schema.TypeString,
					Description:  `What type of storage medium should back this directory. The default is "" which means to use the node's default medium. Must be an empty string (default) or Memory. More info: http://kubernetes.io/docs/user-guide/volumes#emptydir`,
					Optional:     true,
					Default:      "",
					ValidateFunc: validateAttributeValueIsIn([]string{"", "Memory"}),
				},
			},
		},
	}

	v["persistent_volume_claim"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "The specification of a persistent volume.",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"claim_name": {
					Type:        schema.TypeString,
					Description: "ClaimName is the name of a PersistentVolumeClaim in the same ",
					Optional:    true,
				},
				"read_only": {
					Type:        schema.TypeBool,
					Description: "Will force the ReadOnly setting in VolumeMounts.",
					Optional:    true,
					Default:     false,
				},
			},
		},
	}

	v["secret"] = &schema.Schema{
		Type:        schema.TypeList,
		Description: "Secret represents a secret that should populate this volume. More info: http://kubernetes.io/docs/user-guide/volumes#secrets",
		Optional:    true,
		MaxItems:    1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"secret_name": {
					Type:        schema.TypeString,
					Description: "Name of the secret in the pod's namespace to use. More info: http://kubernetes.io/docs/user-guide/volumes#secrets",
					Optional:    true,
				},
			},
		},
	}
	v["name"] = &schema.Schema{
		Type:        schema.TypeString,
		Description: "Volume's name. Must be a DNS_LABEL and unique within the pod. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
		Optional:    true,
	}
	return &schema.Resource{
		Schema: v,
	}
}
