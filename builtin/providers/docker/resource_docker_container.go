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
			"name": &schema.Schema{
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
			"must_run": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  true,
				Optional: true,
			},

			// ForceNew is not true for image because we need to
			// sane this against Docker image IDs, as each image
			// can have multiple names/tags attached do it.
			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"hostname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"domainname": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"command": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"entrypoint": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},

			"dns": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      stringSetHash,
			},

			"publish_all_ports": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"restart": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
				Default:  "no",
				ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
					value := v.(string)
					if !regexp.MustCompile(`^(no|on-failure|always)$`).MatchString(value) {
						es = append(es, fmt.Errorf(
							"%q must be one of \"no\", \"on-failure\", or \"always\"", k))
					}
					return
				},
			},

			"max_retry_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"volumes": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     getVolumesElem(),
				Set:      resourceDockerVolumesHash,
			},

			"ports": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     getPortsElem(),
				Set:      resourceDockerPortsHash,
			},

			"env": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      stringSetHash,
			},

			"links": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Set:      stringSetHash,
			},

			"ip_address": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"ip_prefix_length": &schema.Schema{
				Type:     schema.TypeInt,
				Computed: true,
			},

			"gateway": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"bridge": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},

			"privileged": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},

			"labels": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"memory": &schema.Schema{
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

			"memory_swap": &schema.Schema{
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

			"cpu_shares": &schema.Schema{
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

			"log_driver": &schema.Schema{
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

			"log_opts": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
				ForceNew: true,
			},

			"network_mode": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func getVolumesElem() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"from_container": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"container_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"host_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"read_only": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				ForceNew: true,
			},
		},
	}
}

func getPortsElem() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"internal": &schema.Schema{
				Type:     schema.TypeInt,
				Required: true,
				ForceNew: true,
			},

			"external": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},

			"ip": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				ForceNew: true,
			},

			"protocol": &schema.Schema{
				Type:     schema.TypeString,
				Default:  "tcp",
				Optional: true,
				ForceNew: true,
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

	if v, ok := m["read_only"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(bool)))
	}

	return hashcode.String(buf.String())
}

func stringSetHash(v interface{}) int {
	return hashcode.String(v.(string))
}
