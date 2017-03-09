package dockercloud

import (
	"fmt"
	"regexp"

	"github.com/docker/go-dockercloud/dockercloud"
	"github.com/hashicorp/terraform/helper/schema"
)

func serviceSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"image": &schema.Schema{
			Type:     schema.TypeString,
			Required: true,
		},
		"entrypoint": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"command": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"workdir": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"pid": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
			ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
				value := v.(string)
				if !regexp.MustCompile(`^(none|host)$`).MatchString(value) {
					es = append(es, fmt.Errorf(
						"%q must be one of \"none\" or \"host\"", k))
				}
				return
			},
		},
		"net": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
			ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
				value := v.(string)
				if !regexp.MustCompile(`^(bridge|host)$`).MatchString(value) {
					es = append(es, fmt.Errorf(
						"%q must be one of \"bridge\" or \"host\"", k))
				}
				return
			},
		},
		"deployment_strategy": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
			ForceNew: true,
			ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
				value := v.(string)
				if !regexp.MustCompile(`^(EMPTIEST_NODE|HIGH_AVAILABILITY|EVERY_NODE)$`).MatchString(value) {
					es = append(es, fmt.Errorf(
						"%q must be one of \"EMPTIEST_NODE\", \"HIGH_AVAILABILITY\" or \"EVERY_NODE\"", k))
				}
				return
			},
		},
		"autorestart": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
			ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
				value := v.(string)
				if !regexp.MustCompile(`^(OFF|ON_FAILURE|ALWAYS)$`).MatchString(value) {
					es = append(es, fmt.Errorf(
						"%q must be one of \"OFF\", \"ON_FAILURE\" or \"ALWAYS\"", k))
				}
				return
			},
		},
		"autodestroy": &schema.Schema{
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
			ValidateFunc: func(v interface{}, k string) (ws []string, es []error) {
				value := v.(string)
				if !regexp.MustCompile(`^(OFF|ON_SUCCESS|ALWAYS)$`).MatchString(value) {
					es = append(es, fmt.Errorf(
						"%q must be one of \"OFF\", \"ON_SUCCESS\" or \"ALWAYS\"", k))
				}
				return
			},
		},
		"autoredeploy": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		"privileged": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		"sequential_deployment": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		"container_count": &schema.Schema{
			Type:     schema.TypeInt,
			Optional: true,
			Computed: true,
		},
		"roles": &schema.Schema{
			Type:     schema.TypeList,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"tags": &schema.Schema{
			Type:     schema.TypeList,
			Optional: true,
			Elem:     &schema.Schema{Type: schema.TypeString},
		},
		"bindings": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Set:      bindingsHash,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"container_path": &schema.Schema{
						Type:          schema.TypeString,
						Optional:      true,
						ConflictsWith: []string{"bindings.volumes_from"},
					},
					"host_path": &schema.Schema{
						Type:          schema.TypeString,
						Optional:      true,
						ConflictsWith: []string{"bindings.volumes_from"},
					},
					"rewritable": &schema.Schema{
						Type:     schema.TypeBool,
						Default:  true,
						Optional: true,
					},
					"volumes_from": &schema.Schema{
						Type:          schema.TypeString,
						Optional:      true,
						ConflictsWith: []string{"bindings.container_path"},
					},
				},
			},
		},
		"env": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Set:      envHash,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"key": &schema.Schema{
						Type:     schema.TypeString,
						Required: true,
					},
					"value": &schema.Schema{
						Type:     schema.TypeString,
						Required: true,
					},
				},
			},
		},
		"links": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Set:      linksHash,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"to": &schema.Schema{
						Type:     schema.TypeString,
						Required: true,
					},
					"name": &schema.Schema{
						Type:     schema.TypeString,
						Optional: true,
					},
				},
			},
		},
		"ports": &schema.Schema{
			Type:     schema.TypeSet,
			Optional: true,
			Set:      portsHash,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					"internal": &schema.Schema{
						Type:     schema.TypeInt,
						Required: true,
					},
					"external": &schema.Schema{
						Type:     schema.TypeInt,
						Optional: true,
					},
					"protocol": &schema.Schema{
						Type:     schema.TypeString,
						Default:  "tcp",
						Optional: true,
					},
				},
			},
		},
		"redeploy_on_change": &schema.Schema{
			Type:     schema.TypeBool,
			Optional: true,
		},
		"reuse_existing_volumes": &schema.Schema{
			Type:     schema.TypeBool,
			Default:  true,
			Optional: true,
		},
		"uri": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
		"public_dns": &schema.Schema{
			Type:     schema.TypeString,
			Computed: true,
		},
	}
}

func newServiceCreateRequest(d *schema.ResourceData) dockercloud.ServiceCreateRequest {
	opts := dockercloud.ServiceCreateRequest{
		Name:  d.Get("name").(string),
		Image: d.Get("image").(string),
	}

	if attr, ok := d.GetOk("entrypoint"); ok {
		opts.Entrypoint = attr.(string)
	}

	if attr, ok := d.GetOk("command"); ok {
		opts.Run_command = attr.(string)
	}

	if attr, ok := d.GetOk("workdir"); ok {
		opts.Working_dir = attr.(string)
	}

	if attr, ok := d.GetOk("pid"); ok {
		opts.Pid = attr.(string)
	}

	if attr, ok := d.GetOk("net"); ok {
		opts.Net = attr.(string)
	}

	if attr, ok := d.GetOk("deployment_strategy"); ok {
		opts.Deployment_strategy = attr.(string)
	}

	if attr, ok := d.GetOk("autorestart"); ok {
		opts.Autorestart = attr.(string)
	}

	if attr, ok := d.GetOk("autodestroy"); ok {
		opts.Autodestroy = attr.(string)
	}

	if attr, ok := d.GetOk("autoredeploy"); ok {
		opts.Autoredeploy = attr.(bool)
	}

	if attr, ok := d.GetOk("privileged"); ok {
		opts.Privileged = attr.(bool)
	}

	if attr, ok := d.GetOk("sequential_deployment"); ok {
		opts.Sequential_deployment = attr.(bool)
	}

	if attr, ok := d.GetOk("container_count"); ok {
		opts.Target_num_containers = attr.(int)
	}

	if attr, ok := d.GetOk("roles"); ok {
		opts.Roles = stringListToSlice(attr.([]interface{}))
	}

	if attr, ok := d.GetOk("bindings"); ok {
		opts.Bindings = bindingsSetToServiceBinding(attr.(*schema.Set))
	}

	if attr, ok := d.GetOk("env"); ok {
		opts.Container_envvars = envSetToContainerEnvvar(attr.(*schema.Set))
	}

	if attr, ok := d.GetOk("links"); ok {
		opts.Linked_to_service = linksSetToServiceLinkInfo(attr.(*schema.Set))
	}

	if attr, ok := d.GetOk("ports"); ok {
		opts.Container_ports = portsSetToContainerPortInfo(attr.(*schema.Set))
	}

	if attr, ok := d.GetOk("tags"); ok {
		opts.Tags = stringListToSlice(attr.([]interface{}))
	}

	return opts
}
