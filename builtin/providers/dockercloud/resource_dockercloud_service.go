package dockercloud

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/go-dockercloud/dockercloud"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockercloudService() *schema.Resource {
	return &schema.Resource{
		Create: resourceDockercloudServiceCreate,
		Read:   resourceDockercloudServiceRead,
		Update: resourceDockercloudServiceUpdate,
		Delete: resourceDockercloudServiceDelete,
		Exists: resourceDockercloudServiceExists,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"image": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
				ForceNew: false,
			},
			"entrypoint": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
			"command": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
			"workdir": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
			"pid": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
			"net": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
			"privileged": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
				ForceNew: false,
			},
			"env": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"ports": &schema.Schema{
				Type:     schema.TypeSet,
				Optional: true,
				ForceNew: true,
				Elem: &schema.Resource{
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
				},
			},
			"tags": &schema.Schema{
				Type:     schema.TypeList,
				Optional: true,
				ForceNew: false,
				Elem:     &schema.Schema{Type: schema.TypeString},
			},
			"container_count": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Computed: true,
				ForceNew: false,
			},
			"redeploy_on_change": &schema.Schema{
				Type:     schema.TypeBool,
				Default:  false,
				Optional: true,
				ForceNew: false,
			},
		},
	}
}

func resourceDockercloudServiceCreate(d *schema.ResourceData, meta interface{}) error {
	opts := &dockercloud.ServiceCreateRequest{
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

	if attr, ok := d.GetOk("privileged"); ok {
		opts.Privileged = attr.(bool)
	}

	if attr, ok := d.GetOk("env"); ok {
		opts.Container_envvars = envvarListToContainerEnvvar(attr.(*schema.Set))
	}

	if attr, ok := d.GetOk("ports"); ok {
		opts.Container_ports = portSetToContainerPortInfo(attr.(*schema.Set))
	}

	if attr, ok := d.GetOk("tags"); ok {
		opts.Tags = stringListToStringSlice(attr.([]interface{}))
	}

	if attr, ok := d.GetOk("container_count"); ok {
		opts.Target_num_containers = attr.(int)
	}

	service, err := dockercloud.CreateService(*opts)
	if err != nil {
		return err
	}

	if err = service.Start(); err != nil {
		return fmt.Errorf("Error creating service: %s", err)
	}

	d.SetId(service.Uuid)
	d.Set("state", service.State)

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"Starting"},
		Target:         []string{"Running"},
		Refresh:        newServiceStateRefreshFunc(d, meta),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	serviceRaw, err := stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for service (%s) to become ready: %s", d.Id(), err)
	}

	service = serviceRaw.(dockercloud.Service)
	d.Set("state", service.State)

	return resourceDockercloudServiceRead(d, meta)
}

func resourceDockercloudServiceRead(d *schema.ResourceData, meta interface{}) error {
	service, err := dockercloud.GetService(d.Id())
	if err != nil {
		if strings.Contains(err.Error(), "404 NOT FOUND") {
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Error retrieving service: %s", err)
	}

	if service.State == "Terminated" {
		d.SetId("")
		return nil
	}

	d.Set("name", service.Name)
	d.Set("image", service.Image_name)
	d.Set("entrypoint", service.Entrypoint)
	d.Set("command", service.Run_command)
	d.Set("workdir", service.Working_dir)
	d.Set("pid", service.Pid)
	d.Set("net", service.Net)
	d.Set("privileged", service.Privileged)
	d.Set("env", containerEnvvarsToList(service.Container_envvars))
	d.Set("tags", serviceTagsToList(service.Tags))
	d.Set("container_count", service.Target_num_containers)
	d.Set("state", service.State)

	return nil
}

func resourceDockercloudServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	var change bool

	opts := &dockercloud.ServiceCreateRequest{}

	if d.HasChange("image") {
		_, newImage := d.GetChange("image")
		opts.Image = newImage.(string)
		change = true
	}

	if d.HasChange("entrypoint") {
		_, newEntrypoint := d.GetChange("entrypoint")
		opts.Entrypoint = newEntrypoint.(string)
		change = true
	}

	if d.HasChange("command") {
		_, newCommand := d.GetChange("command")
		opts.Run_command = newCommand.(string)
		change = true
	}

	if d.HasChange("workdir") {
		_, newWorkdir := d.GetChange("workdir")
		opts.Working_dir = newWorkdir.(string)
		change = true
	}

	if d.HasChange("pid") {
		_, newPid := d.GetChange("pid")
		opts.Pid = newPid.(string)
		change = true
	}

	if d.HasChange("net") {
		_, newNet := d.GetChange("net")
		opts.Net = newNet.(string)
		change = true
	}

	if d.HasChange("privileged") {
		_, newPrivileged := d.GetChange("privileged")
		opts.Privileged = newPrivileged.(bool)
		change = true
	}

	if d.HasChange("env") {
		_, newEnvvars := d.GetChange("env")
		envvars := newEnvvars.(*schema.Set)
		opts.Container_envvars = envvarListToContainerEnvvar(envvars)
		change = true
	}

	if d.HasChange("ports") {
		_, newPorts := d.GetChange("ports")
		ports := newPorts.(*schema.Set)
		opts.Container_ports = portSetToContainerPortInfo(ports)
		change = true
	}

	if d.HasChange("tags") {
		_, newTags := d.GetChange("tags")
		tags := newTags.([]interface{})
		opts.Tags = stringListToStringSlice(tags)
	}

	if d.HasChange("container_count") {
		_, newNum := d.GetChange("container_count")
		opts.Target_num_containers = newNum.(int)
	}

	service, err := dockercloud.GetService(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving service (%s): %s", d.Id(), err)
	}

	if err := service.Update(*opts); err != nil {
		return fmt.Errorf("Error updating service: %s", err)
	}

	if d.Get("redeploy_on_change").(bool) && change {
		if err := service.Redeploy(dockercloud.ReuseVolumesOption{Reuse: true}); err != nil {
			return fmt.Errorf("Error redeploying containers: %s", err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:        []string{"Redeploying"},
			Target:         []string{"Running"},
			Refresh:        newServiceStateRefreshFunc(d, meta),
			Timeout:        60 * time.Minute,
			Delay:          10 * time.Second,
			MinTimeout:     3 * time.Second,
			NotFoundChecks: 60,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for service (%s) to finish scaling: %s", d.Id(), err)
		}
	}

	if d.HasChange("container_count") {
		if err := service.Scale(); err != nil {
			return fmt.Errorf("Error updating service: %s", err)
		}

		stateConf := &resource.StateChangeConf{
			Pending:        []string{"Scaling"},
			Target:         []string{"Running"},
			Refresh:        newServiceStateRefreshFunc(d, meta),
			Timeout:        60 * time.Minute,
			Delay:          10 * time.Second,
			MinTimeout:     3 * time.Second,
			NotFoundChecks: 60,
		}

		_, err = stateConf.WaitForState()
		if err != nil {
			return fmt.Errorf("Error waiting for service (%s) to finish scaling: %s", d.Id(), err)
		}
	}

	return nil
}

func resourceDockercloudServiceDelete(d *schema.ResourceData, meta interface{}) error {
	service, err := dockercloud.GetService(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving service (%s): %s", d.Id(), err)
	}

	if service.State == "Terminated" {
		d.SetId("")
		return nil
	}

	if err = service.TerminateService(); err != nil {
		return fmt.Errorf("Error deleting service (%s): %s", d.Id(), err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"Terminating", "Stopped"},
		Target:         []string{"Terminated"},
		Refresh:        newServiceStateRefreshFunc(d, meta),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for service (%s) to terminate: %s", d.Id(), err)
	}

	d.SetId("")

	return nil
}

func resourceDockercloudServiceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	service, err := dockercloud.GetService(d.Id())
	if err != nil {
		return false, err
	}

	if service.Uuid == d.Id() {
		return true, nil
	}

	return false, nil
}

func newServiceStateRefreshFunc(d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		service, err := dockercloud.GetService(d.Id())
		if err != nil {
			return nil, "", err
		}

		if service.State == "Stopped" {
			return nil, "", fmt.Errorf("Service entered 'Stopped' state")
		}

		return service, service.State, nil
	}
}

func portSetToContainerPortInfo(ports *schema.Set) []dockercloud.ContainerPortInfo {
	containerPorts := []dockercloud.ContainerPortInfo{}
	for _, portInt := range ports.List() {
		containerPortInfo := dockercloud.ContainerPortInfo{}
		port := portInt.(map[string]interface{})

		containerPortInfo.Inner_port = port["internal"].(int)
		containerPortInfo.Protocol = port["protocol"].(string)

		external, extOk := port["external"].(int)
		if extOk {
			containerPortInfo.Outer_port = external
		}

		containerPorts = append(containerPorts, containerPortInfo)
	}

	return containerPorts
}

func envvarListToContainerEnvvar(envvars *schema.Set) []dockercloud.ContainerEnvvar {
	containerEnvvars := []dockercloud.ContainerEnvvar{}
	for _, e := range envvars.List() {
		envvar := strings.Split(e.(string), "=")
		containerEnvvar := dockercloud.ContainerEnvvar{
			Key:   envvar[0],
			Value: envvar[1],
		}
		containerEnvvars = append(containerEnvvars, containerEnvvar)
	}
	return containerEnvvars
}

func containerEnvvarsToList(envvars []dockercloud.ContainerEnvvar) []string {
	ret := []string{}
	for _, envvar := range envvars {
		envvarStr := fmt.Sprintf("%s=%s", envvar.Key, envvar.Value)
		ret = append(ret, envvarStr)
	}
	return ret
}

func serviceTagsToList(tags []dockercloud.ServiceTag) []string {
	ret := []string{}
	for _, tag := range tags {
		ret = append(ret, tag.Name)
	}
	return ret
}

func stringListToStringSlice(stringList []interface{}) []string {
	ret := []string{}
	for _, v := range stringList {
		ret = append(ret, v.(string))
	}
	return ret
}
