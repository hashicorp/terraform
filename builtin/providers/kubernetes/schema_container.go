package kubernetes

import "github.com/hashicorp/terraform/helper/schema"

func handlerFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"exec": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "exec specifies the action to take.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"command": {
						Type:        schema.TypeList,
						Description: `Command is the command line to execute inside the container, the working directory for the command is root ('/') in the container's filesystem. The command is simply exec'd, it is not run inside a shell, so traditional shell instructions. To use a shell, you need to explicitly call out to that shell. Exit status of 0 is treated as live/healthy and non-zero is unhealthy.`,
						Optional:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
					},
				},
			},
		},
		"http_get": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "Specifies the http request to perform.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"host": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: `Host name to connect to, defaults to the pod IP. You probably want to set "Host" in httpHeaders instead.`,
					},
					"path": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: `Path to access on the HTTP server.`,
					},
					"scheme": {
						Type:        schema.TypeString,
						Optional:    true,
						Default:     "HTTP",
						Description: `Scheme to use for connecting to the host.`,
					},
					"port": {
						Type:         schema.TypeString,
						Optional:     true,
						ValidateFunc: validatePortNumOrName,
						Description:  `Name or number of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.`,
					},
					"http_header": {
						Type:        schema.TypeList,
						Optional:    true,
						Description: `Scheme to use for connecting to the host.`,
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"name": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "The header field name",
								},
								"value": {
									Type:        schema.TypeString,
									Optional:    true,
									Description: "The header field value",
								},
							},
						},
					},
				},
			},
		},
		"tcp_socket": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "TCPSocket specifies an action involving a TCP port. TCP hooks not yet supported",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"port": {
						Type:         schema.TypeString,
						Required:     true,
						ValidateFunc: validatePortNumOrName,
						Description:  "Number or name of the port to access on the container. Number must be in the range 1 to 65535. Name must be an IANA_SVC_NAME.",
					},
				},
			},
		},
	}
}

func resourcesField() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"limits": {
			Type:        schema.TypeList,
			Optional:    true,
			Computed:    true,
			MaxItems:    1,
			Description: "Describes the maximum amount of compute resources allowed. More info: http://kubernetes.io/docs/user-guide/compute-resources/",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"cpu": {
						Type:             schema.TypeString,
						Optional:         true,
						Computed:         true,
						ValidateFunc:     validateResourceQuantity,
						DiffSuppressFunc: suppressEquivalentResourceQuantity,
					},
					"memory": {
						Type:             schema.TypeString,
						Optional:         true,
						Computed:         true,
						ValidateFunc:     validateResourceQuantity,
						DiffSuppressFunc: suppressEquivalentResourceQuantity,
					},
				},
			},
		},
		"requests": {
			Type:     schema.TypeList,
			Optional: true,
			Computed: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"cpu": {
						Type:             schema.TypeString,
						Optional:         true,
						Computed:         true,
						ValidateFunc:     validateResourceQuantity,
						DiffSuppressFunc: suppressEquivalentResourceQuantity,
					},
					"memory": {
						Type:             schema.TypeString,
						Optional:         true,
						Computed:         true,
						ValidateFunc:     validateResourceQuantity,
						DiffSuppressFunc: suppressEquivalentResourceQuantity,
					},
				},
			},
		},
	}
}

func seLinuxOptionsField() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"level": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Level is SELinux level label that applies to the container.",
		},
		"role": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Role is a SELinux role label that applies to the container.",
		},
		"type": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Type is a SELinux type label that applies to the container.",
		},
		"user": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "User is a SELinux user label that applies to the container.",
		},
	}
}

func volumeMountFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"mount_path": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Path within the container at which the volume should be mounted. Must not contain ':'.",
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "This must match the Name of a Volume.",
		},
		"read_only": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Mounted read-only if true, read-write otherwise (false or unspecified). Defaults to false.",
		},
		"sub_path": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: `Path within the volume from which the container's volume should be mounted. Defaults to "" (volume's root).`,
		},
	}
}

func containerFields() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"args": {
			Type:        schema.TypeList,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Arguments to the entrypoint. The docker image's CMD is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/containers#containers-and-commands",
		},
		"command": {
			Type:        schema.TypeList,
			Optional:    true,
			Elem:        &schema.Schema{Type: schema.TypeString},
			Description: "Entrypoint array. Not executed within a shell. The docker image's ENTRYPOINT is used if this is not provided. Variable references $(VAR_NAME) are expanded using the container's environment. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/containers#containers-and-commands",
		},
		"env": {
			Type:        schema.TypeList,
			Optional:    true,
			ForceNew:    true,
			Description: "List of environment variables to set in the container. Cannot be updated.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"name": {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Name of the environment variable. Must be a C_IDENTIFIER",
					},
					"value": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: `Variable references $(VAR_NAME) are expanded using the previous defined environment variables in the container and any service environment variables. If a variable cannot be resolved, the reference in the input string will be unchanged. The $(VAR_NAME) syntax can be escaped with a double $$, ie: $$(VAR_NAME). Escaped references will never be expanded, regardless of whether the variable exists or not. Defaults to "".`,
					},
					"value_from": {
						Type:        schema.TypeList,
						Optional:    true,
						MaxItems:    1,
						Description: "Source for the environment variable's value",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								"config_map_key_ref": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "Selects a key of a ConfigMap.",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"key": {
												Type:        schema.TypeString,
												Optional:    true,
												Description: "The key to select.",
											},
											"name": {
												Type:        schema.TypeString,
												Optional:    true,
												Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
											},
										},
									},
								},
								"field_ref": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations, spec.nodeName, spec.serviceAccountName, status.podIP..",
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
								"resource_field_ref": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations, spec.nodeName, spec.serviceAccountName, status.podIP..",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"container_name": {
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
								"secret_key_ref": {
									Type:        schema.TypeList,
									Optional:    true,
									MaxItems:    1,
									Description: "Selects a field of the pod: supports metadata.name, metadata.namespace, metadata.labels, metadata.annotations, spec.nodeName, spec.serviceAccountName, status.podIP..",
									Elem: &schema.Resource{
										Schema: map[string]*schema.Schema{
											"key": {
												Type:        schema.TypeString,
												Optional:    true,
												Description: "The key of the secret to select from. Must be a valid secret key.",
											},
											"name": {
												Type:        schema.TypeString,
												Optional:    true,
												Description: "Name of the referent. More info: http://kubernetes.io/docs/user-guide/identifiers#names",
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"image": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Docker image name. More info: http://kubernetes.io/docs/user-guide/images",
		},
		"image_pull_policy": {
			Type:        schema.TypeString,
			Optional:    true,
			Computed:    true,
			Description: "Image pull policy. One of Always, Never, IfNotPresent. Defaults to Always if :latest tag is specified, or IfNotPresent otherwise. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/images#updating-images",
		},
		"lifecycle": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			ForceNew:    true,
			Description: "Actions that the management system should take in response to container lifecycle events",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"post_start": {
						Type:        schema.TypeList,
						Description: `post_start is called immediately after a container is created. If the handler fails, the container is terminated and restarted according to its restart policy. Other management of the container blocks until the hook completes. More info: http://kubernetes.io/docs/user-guide/container-environment#hook-details`,
						Optional:    true,
						Elem: &schema.Resource{
							Schema: handlerFields(),
						},
					},
					"pre_stop": {
						Type:        schema.TypeList,
						Description: `pre_stop is called immediately before a container is terminated. The container is terminated after the handler completes. The reason for termination is passed to the handler. Regardless of the outcome of the handler, the container is eventually terminated. Other management of the container blocks until the hook completes. More info: http://kubernetes.io/docs/user-guide/container-environment#hook-details`,
						Optional:    true,
						Elem: &schema.Resource{
							Schema: handlerFields(),
						},
					},
				},
			},
		},
		"liveness_probe": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			ForceNew:    true,
			Description: "Periodic probe of container liveness. Container will be restarted if the probe fails. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/pod-states#container-probes",
			Elem:        probeSchema(),
		},
		"name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Name of the container specified as a DNS_LABEL. Each container in a pod must have a unique name (DNS_LABEL). Cannot be updated.",
		},
		"port": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: `List of ports to expose from the container. Exposing a port here gives the system additional information about the network connections a container uses, but is primarily informational. Not specifying a port here DOES NOT prevent that port from being exposed. Any port which is listening on the default "0.0.0.0" address inside a container will be accessible from the network. Cannot be updated.`,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"container_port": {
						Type:         schema.TypeInt,
						Required:     true,
						ValidateFunc: validatePortNumOrName,
						Description:  "Number of port to expose on the pod's IP address. This must be a valid port number, 0 < x < 65536.",
					},
					"host_ip": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "What host IP to bind the external port to.",
					},
					"host_port": {
						Type:        schema.TypeInt,
						Optional:    true,
						Description: "Number of port to expose on the host. If specified, this must be a valid port number, 0 < x < 65536. If HostNetwork is specified, this must match ContainerPort. Most containers do not need this.",
					},
					"name": {
						Type:         schema.TypeString,
						Optional:     true,
						ValidateFunc: validatePortNumOrName,
						Description:  "If specified, this must be an IANA_SVC_NAME and unique within the pod. Each named port in a pod must have a unique name. Name for the port that can be referred to by services",
					},
					"protocol": {
						Type:        schema.TypeString,
						Optional:    true,
						Description: `Protocol for port. Must be UDP or TCP. Defaults to "TCP".`,
						Default:     "TCP",
					},
				},
			},
		},
		"readiness_probe": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			ForceNew:    true,
			Description: "Periodic probe of container service readiness. Container will be removed from service endpoints if the probe fails. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/pod-states#container-probes",
			Elem:        probeSchema(),
		},
		"resources": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Computed:    true,
			Description: "Compute Resources required by this container. Cannot be updated. More info: http://kubernetes.io/docs/user-guide/persistent-volumes#resources",
			Elem: &schema.Resource{
				Schema: resourcesField(),
			},
		},

		"security_context": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			ForceNew:    true,
			Description: "Security options the pod should run with. More info: http://releases.k8s.io/HEAD/docs/design/security_context.md",
			Elem:        securityContextSchema(),
		},
		"stdin": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Whether this container should allocate a buffer for stdin in the container runtime. If this is not set, reads from stdin in the container will always result in EOF. ",
		},
		"stdin_once": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Whether the container runtime should close the stdin channel after it has been opened by a single attach. When stdin is true the stdin stream will remain open across multiple attach sessions. If stdinOnce is set to true, stdin is opened on container start, is empty until the first client attaches to stdin, and then remains open and accepts data until the client disconnects, at which time stdin is closed and remains closed until the container is restarted. If this flag is false, a container processes that reads from stdin will never receive an EOF.",
		},
		"termination_message_path": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Default:     "/dev/termination-log",
			Description: "Optional: Path at which the file to which the container's termination message will be written is mounted into the container's filesystem. Message written is intended to be brief final status, such as an assertion failure message. Defaults to /dev/termination-log. Cannot be updated.",
		},
		"tty": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Whether this container should allocate a TTY for itself",
		},
		"volume_mount": {
			Type:        schema.TypeList,
			Optional:    true,
			Description: "Pod volumes to mount into the container's filesystem. Cannot be updated.",
			Elem: &schema.Resource{
				Schema: volumeMountFields(),
			},
		},
		"working_dir": {
			Type:        schema.TypeString,
			Optional:    true,
			ForceNew:    true,
			Description: "Container's working directory. If not specified, the container runtime's default will be used, which might be configured in the container image. Cannot be updated.",
		},
	}
}

func probeSchema() *schema.Resource {
	h := handlerFields()
	h["failure_threshold"] = &schema.Schema{
		Type:         schema.TypeInt,
		Optional:     true,
		Description:  "Minimum consecutive failures for the probe to be considered failed after having succeeded.",
		Default:      3,
		ValidateFunc: validatePositiveInteger,
	}
	h["initial_delay_seconds"] = &schema.Schema{
		Type:        schema.TypeInt,
		Optional:    true,
		Description: "Number of seconds after the container has started before liveness probes are initiated. More info: http://kubernetes.io/docs/user-guide/pod-states#container-probes",
	}
	h["period_seconds"] = &schema.Schema{
		Type:         schema.TypeInt,
		Optional:     true,
		Default:      10,
		ValidateFunc: validatePositiveInteger,
		Description:  "How often (in seconds) to perform the probe",
	}
	h["success_threshold"] = &schema.Schema{
		Type:         schema.TypeInt,
		Optional:     true,
		Default:      1,
		ValidateFunc: validatePositiveInteger,
		Description:  "Minimum consecutive successes for the probe to be considered successful after having failed.",
	}

	h["timeout_seconds"] = &schema.Schema{
		Type:         schema.TypeInt,
		Optional:     true,
		Default:      1,
		ValidateFunc: validatePositiveInteger,
		Description:  "Number of seconds after which the probe times out. More info: http://kubernetes.io/docs/user-guide/pod-states#container-probes",
	}
	return &schema.Resource{
		Schema: h,
	}

}

func securityContextSchema() *schema.Resource {
	m := map[string]*schema.Schema{
		"privileged": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: `Run container in privileged mode. Processes in privileged containers are essentially equivalent to root on the host.`,
		},
		"read_only_root_filesystem": {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Whether this container has a read-only root filesystem.",
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
		"se_linux_options": {
			Type:        schema.TypeList,
			Description: "ImagePullSecrets is an optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec. If specified, these secrets will be passed to individual puller implementations for them to use. For example, in the case of docker, only DockerConfig type secrets are honored. More info: http://kubernetes.io/docs/user-guide/images#specifying-imagepullsecrets-on-a-pod",
			Optional:    true,
			MaxItems:    1,
			Elem: &schema.Resource{
				Schema: seLinuxOptionsField(),
			},
		},
		"capabilities": {
			Type:        schema.TypeList,
			Optional:    true,
			MaxItems:    1,
			Description: "The capabilities to add/drop when running containers. Defaults to the default set of capabilities granted by the container runtime.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"add": {
						Type:        schema.TypeList,
						Optional:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Description: "Added capabilities",
					},
					"drop": {
						Type:        schema.TypeList,
						Optional:    true,
						Elem:        &schema.Schema{Type: schema.TypeString},
						Description: "Removed capabilities",
					},
				},
			},
		},
	}

	return &schema.Resource{
		Schema: m,
	}
}
