package docker

import (
	"bytes"
	"fmt"

	"regexp"

	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerContainer() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockerContainerCreate,
		Read:   resourceDockerContainerRead,
		Update: resourceDockerContainerUpdate,
		Delete: resourceDockerContainerDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			// Indicates whether the container must be running.
			//
			// An assumption is made that configured containers
			// should be running; if not, they should not be in
			// the configuration. Therefore a stopped container
			// should be started. Set to false to have the
			// provider leave the container alone.
			//
			// Actively-debugged containers are likely to be
			// stopped and started manually, and Docker has
			// some provisions for restarting containers that
			// stop. The utility here comes from the fact that
			// this will delete and re-create the container
			// following the principle that the containers
			// should be pristine when started.
			"must_run": {
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},

			// ForceNew is not true for image because we need to
			// sane this against Docker image IDs, as each image
			// can have multiple names/tags attached do it.
			"image": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"hostname": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"domainname": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"command": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"entrypoint": {
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"user": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"dns": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"dns_opts": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"dns_search": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"publish_all_ports": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"restart": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "no",
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !regexp.MustCompile(`^(no|on-failure|always|unless-stopped)$`).MatchString(value) {
						es = append(es, fmt.Errorf(
							"%q must be one of \"no\", \"on-failure\", \"always\" or \"unless-stopped\"", k))
					}
					return
				},
			},

			"max_retry_count": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"volumes": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"from_container": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"container_path": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"host_path": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
							ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
								value := v.(string)
								if !regexp.MustCompile(`^/`).MatchString(value) {
									es = append(es, fmt.Errorf(
										"%q must be an absolute path", k))
								}
								return
							},
						},

						"volume_name": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"read_only": {
							Type:     schema.TypeBool,
							Optional: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceDockerVolumesHash,
			},

			"ports": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"internal": {
							Type:     schema.TypeInt,
							Required: true,
							ForceNew: true,
						},

						"external": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},

						"ip": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},

						"protocol": {
							Type:     schema.TypeString,
							Default:  "tcp",
							Optional: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceDockerPortsHash,
			},

			"host": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ip": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},

						"host": {
							Type:     schema.TypeString,
							Required: true,
							ForceNew: true,
						},
					},
				},
				Set: resourceDockerHostsHash,
			},

			"env": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"links": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},

			"ip_address": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"ip_prefix_length": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"gateway": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"bridge": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"privileged": {
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"destroy_grace_seconds": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"labels": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"memory": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(int)
					if value < 0 {
						es = append(es, fmt.Errorf("%q must be greater than or equal to 0", k))
					}
					return
				},
			},

			"memory_swap": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(int)
					if value < -1 {
						es = append(es, fmt.Errorf("%q must be greater than or equal to -1", k))
					}
					return
				},
			},

			"cpu_shares": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(int)
					if value < 0 {
						es = append(es, fmt.Errorf("%q must be greater than or equal to 0", k))
					}
					return
				},
			},

			"log_driver": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "json-file",
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !regexp.MustCompile(`^(json-file|syslog|journald|gelf|fluentd)$`).MatchString(value) {
						es = append(es, fmt.Errorf(
							"%q must be one of \"json-file\", \"syslog\", \"journald\", \"gelf\", or \"fluentd\"", k))
					}
					return
				},
			},

			"log_opts": {
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"network_mode": {
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"networks": {
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      schema.HashString,
			},
		},
	}
}

func resourceDockerPortsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%v-", m["internal"].(int)))

	if v, ok := m["external"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(int)))
	}

	if v, ok := m["ip"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["protocol"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func resourceDockerHostsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["ip"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["host"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func resourceDockerVolumesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["from_container"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["container_path"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["host_path"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["volume_name"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["read_only"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(bool)))
	}

	return hashcode.String(buf.String())
}
