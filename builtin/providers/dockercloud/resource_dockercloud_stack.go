package dockercloud

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/docker/go-dockercloud/dockercloud"
	"github.com/hashicorp/terraform/helper/hashcode"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockercloudStack() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockercloudStackCreate,
		Read:   resourceDockercloudStackRead,
		Update: resourceDockercloudStackUpdate,
		Delete: resourceDockercloudStackDelete,
		Exists: resourceDockercloudStackExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"services": &schema.Schema{
				Type:     schema.TypeSet,
				Required: true,
				Set:      servicesHash,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
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
										"%q must be one of \"off\", \"on-failure\" or \"always\"", k))
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
										"%q must be one of \"off\", \"on-success\" or \"always\"", k))
								}
								return
							},
						},
						"autoredeploy": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},
						"privileged": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},
						"sequential_deployment": &schema.Schema{
							Type:     schema.TypeBool,
							Optional: true,
							Computed: true,
						},
						"container_count": &schema.Schema{
							Type:     schema.TypeInt,
							Optional: true,
							Computed: true,
						},
						"roles": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"tags": &schema.Schema{
							Type:     schema.TypeList,
							Optional: true,
							Computed: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
						},
						"bindings": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
							Set:      bindingsHash,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"container_path": &schema.Schema{
										Type:          schema.TypeString,
										Optional:      true,
										ConflictsWith: []string{"services.bindings.volumes_from"},
									},
									"host_path": &schema.Schema{
										Type:          schema.TypeString,
										Optional:      true,
										ConflictsWith: []string{"services.bindings.volumes_from"},
									},
									"rewritable": &schema.Schema{
										Type:     schema.TypeBool,
										Optional: true,
										Computed: true,
									},
									"volumes_from": &schema.Schema{
										Type:          schema.TypeString,
										Optional:      true,
										ConflictsWith: []string{"services.bindings.container_path"},
									},
								},
							},
						},
						"env": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
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
							Computed: true,
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
										Computed: true,
									},
								},
							},
						},
						"ports": &schema.Schema{
							Type:     schema.TypeSet,
							Optional: true,
							Computed: true,
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
										Computed: true,
									},
									"protocol": &schema.Schema{
										Type:     schema.TypeString,
										Default:  "tcp",
										Optional: true,
									},
								},
							},
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
		},
	}
}

func resourceDockercloudStackCreate(d *schema.ResourceData, meta interface{}) error {
	opts := dockercloud.StackCreateRequest{
		Name: d.Get("name").(string),
	}

	services := d.Get("services").(*schema.Set)

	serviceCreateRequests := make([]dockercloud.ServiceCreateRequest, services.Len())

	for i, s := range services.List() {
		service := s.(map[string]interface{})

		serviceCreateRequests[i] = dockercloud.ServiceCreateRequest{
			Name:                  service["name"].(string),
			Image:                 service["image"].(string),
			Entrypoint:            service["entrypoint"].(string),
			Run_command:           service["command"].(string),
			Working_dir:           service["workdir"].(string),
			Pid:                   service["pid"].(string),
			Net:                   service["net"].(string),
			Deployment_strategy:   service["deployment_strategy"].(string),
			Autorestart:           service["autorestart"].(string),
			Autodestroy:           service["autodestroy"].(string),
			Autoredeploy:          service["autoredeploy"].(bool),
			Privileged:            service["privileged"].(bool),
			Sequential_deployment: service["sequential_deployment"].(bool),
			Target_num_containers: service["container_count"].(int),
			Roles:             stringListToSlice(service["roles"].([]interface{})),
			Tags:              stringListToSlice(service["tags"].([]interface{})),
			Bindings:          bindingsSetToServiceBinding(service["bindings"].(*schema.Set)),
			Container_envvars: envSetToContainerEnvvar(service["env"].(*schema.Set)),
			Linked_to_service: linksSetToServiceLinkInfo(service["links"].(*schema.Set)),
			Container_ports:   portsSetToContainerPortInfo(service["ports"].(*schema.Set)),
		}
	}

	opts.Services = serviceCreateRequests

	stack, err := dockercloud.CreateStack(opts)
	if err != nil {
		if strings.Contains(err.Error(), "409 CONFLICT") {
			return fmt.Errorf("Duplicate stack name: %s", opts.Name)
		}
		return err
	}

	d.SetId(stack.Uuid)

	if err = stack.Start(); err != nil {
		return fmt.Errorf("Error starting stack: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"Starting"},
		Target:         []string{"Running"},
		Refresh:        newStackStateRefreshFunc(d, meta),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	stackRaw, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for stack (%s) to become ready: %s", d.Id(), err)
	}

	stack = stackRaw.(dockercloud.Stack)
	d.Set("uri", stack.Resource_uri)

	return resourceDockercloudStackRead(d, meta)
}

func resourceDockercloudStackRead(d *schema.ResourceData, meta interface{}) error {
	stack, err := dockercloud.GetStack(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404 NOT FOUND") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving stack: %s", err)
	}

	if stack.State == "Terminated" {
		d.SetId("")
		return nil
	}

	services, err := flattenServices(stack.Services)
	if err != nil {
		return fmt.Errorf("Could not get stack service: %s", err.Error())
	}

	d.Set("services", services)
	d.Set("uri", stack.Resource_uri)

	return nil
}

func resourceDockercloudStackUpdate(d *schema.ResourceData, meta interface{}) error {
	var change bool
	var opts dockercloud.StackCreateRequest

	if d.HasChange("name") {
		_, newName := d.GetChange("name")
		opts.Name = newName.(string)
		change = true
	}

	if d.HasChange("services") {
		_, newServices := d.GetChange("services")

		serviceCreateRequests := make([]dockercloud.ServiceCreateRequest, newServices.(*schema.Set).Len())

		for i, s := range newServices.(*schema.Set).List() {
			service := s.(map[string]interface{})

			serviceCreateRequests[i] = dockercloud.ServiceCreateRequest{
				Name:                  service["name"].(string),
				Image:                 service["image"].(string),
				Entrypoint:            service["entrypoint"].(string),
				Run_command:           service["command"].(string),
				Working_dir:           service["workdir"].(string),
				Pid:                   service["pid"].(string),
				Net:                   service["net"].(string),
				Deployment_strategy:   service["deployment_strategy"].(string),
				Autorestart:           service["autorestart"].(string),
				Autodestroy:           service["autodestroy"].(string),
				Autoredeploy:          service["autoredeploy"].(bool),
				Privileged:            service["privileged"].(bool),
				Sequential_deployment: service["sequential_deployment"].(bool),
				Roles:                 stringListToSlice(service["roles"].([]interface{})),
				Tags:                  stringListToSlice(service["tags"].([]interface{})),
				Bindings:              bindingsSetToServiceBinding(service["bindings"].(*schema.Set)),
				Container_envvars:     envSetToContainerEnvvar(service["env"].(*schema.Set)),
				Linked_to_service:     linksSetToServiceLinkInfo(service["links"].(*schema.Set)),
				Container_ports:       portsSetToContainerPortInfo(service["ports"].(*schema.Set)),
				Target_num_containers: service["container_count"].(int),
			}
		}

		opts.Services = serviceCreateRequests
		change = true
	}

	stack, err := dockercloud.GetStack(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving stack (%s): %s", d.Id(), err)
	}

	if err := stack.Update(opts); err != nil {
		return fmt.Errorf("Error updating stack: %s", err)
	}

	if d.Get("redeploy_on_change").(bool) && change {
		reuse := d.Get("reuse_existing_volumes").(bool)
		if err := stack.Redeploy(dockercloud.ReuseVolumesOption{Reuse: reuse}); err != nil {
			return fmt.Errorf("Error redeploying stack: %s", err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:        []string{"Redeploying"},
			Target:         []string{"Running"},
			Refresh:        newStackStateRefreshFunc(d, meta),
			Timeout:        60 * time.Minute,
			Delay:          10 * time.Second,
			MinTimeout:     3 * time.Second,
			NotFoundChecks: 60,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for stack (%s) to finish scaling: %s", d.Id(), err)
		}
	}

	return resourceDockercloudStackRead(d, meta)
}

func resourceDockercloudStackDelete(d *schema.ResourceData, meta interface{}) error {
	stack, err := dockercloud.GetStack(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving stack (%s): %s", d.Id(), err)
	}

	if stack.State == "Terminated" {
		d.SetId("")
		return nil
	}

	if err = stack.Terminate(); err != nil {
		return fmt.Errorf("Error deleting stack (%s): %s", d.Id(), err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"Terminating", "Stopped"},
		Target:         []string{"Terminated"},
		Refresh:        newStackStateRefreshFunc(d, meta),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for stack (%s) to terminate: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}

func resourceDockercloudStackExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	stack, err := dockercloud.GetStack(d.Id())
	if err != nil {
		return false, err
	}

	if stack.Uuid == d.Id() {
		return true, nil
	}

	return false, nil
}

func newStackStateRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		stack, err := dockercloud.GetStack(d.Id())
		if err != nil {
			return nil, "", err
		}

		if stack.State == "Stopped" {
			return nil, "", fmt.Errorf("Stack entered 'Stopped' state")
		}

		return stack, stack.State, nil
	}
}

func flattenServices(s []string) (*schema.Set, error) {
	services := make([]interface{}, len(s))

	for i, sid := range s {
		service, err := dockercloud.GetService(sid)
		if err != nil {
			return &schema.Set{}, err
		}

		if service.State == "Terminated" {
			continue
		}

		services[i] = map[string]interface{}{
			"name":       service.Name,
			"image":      service.Image_name,
			"entrypoint": service.Entrypoint,
			"command":    service.Run_command,
			"workdir":    service.Working_dir,
			"pid":        service.Pid,
			"net":        service.Net,
			"deployment_strategy":   service.Deployment_strategy,
			"autorestart":           service.Autorestart,
			"autodestroy":           service.Autodestroy,
			"autoredeploy":          service.Autoredeploy,
			"privileged":            service.Privileged,
			"sequential_deployment": service.Sequential_deployment,
			"container_count":       service.Target_num_containers,
			"roles":                 stringSliceToList(service.Roles),
			"tags":                  stringSliceToList(flattenTags(service.Tags)),
			"bindings":              flattenBindings(service.Bindings),
			"env":                   flattenEnv(service.Container_envvars),
			"links":                 flattenLinks(service.Linked_to_service),
			"ports":                 flattenPorts(service.Container_ports),
		}
	}

	return schema.NewSet(servicesHash, services), nil
}

func servicesHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s", m["name"].(string)))
	buf.WriteString(fmt.Sprintf("%v-", m["image"].(string)))

	if v, ok := m["entrypoint"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["command"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["workdir"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["pid"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["net"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["deployment_strategy"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["autorestart"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["autodestroy"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["autoredeploy"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(bool)))
	}

	if v, ok := m["privileged"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(bool)))
	}

	if v, ok := m["sequential_deployment"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(bool)))
	}

	if v, ok := m["container_count"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(int)))
	}

	if v, ok := m["roles"]; ok {
		for _, x := range v.([]interface{}) {
			buf.WriteString(fmt.Sprintf("%v-", x.(string)))
		}
	}

	if v, ok := m["tags"]; ok {
		for _, x := range v.([]interface{}) {
			buf.WriteString(fmt.Sprintf("%v-", x.(string)))
		}
	}

	if v, ok := m["bindings"]; ok {
		for _, x := range v.(*schema.Set).List() {
			buf.WriteString(fmt.Sprintf("%d-", bindingsHash(x.(map[string]interface{}))))
		}
	}

	if v, ok := m["env"]; ok {
		for _, x := range v.(*schema.Set).List() {
			buf.WriteString(fmt.Sprintf("%d-", envHash(x.(map[string]interface{}))))
		}
	}

	if v, ok := m["links"]; ok {
		for _, x := range v.(*schema.Set).List() {
			buf.WriteString(fmt.Sprintf("%d-", linksHash(x.(map[string]interface{}))))
		}
	}

	if v, ok := m["ports"]; ok {
		for _, x := range v.(*schema.Set).List() {
			buf.WriteString(fmt.Sprintf("%d-", portsHash(x.(map[string]interface{}))))
		}
	}

	return hashcode.String(buf.String())
}
