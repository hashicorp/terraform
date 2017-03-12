package dockercloud

import (
	"bytes"
	"fmt"
	"time"

	"github.com/docker/go-dockercloud/dockercloud"
	"github.com/hashicorp/terraform/helper/hashcode"
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
		Schema: serviceSchema(),
	}
}

func resourceDockercloudServiceCreate(d *schema.ResourceData, meta interface{}) error {
	opts := newServiceCreateRequest(d)
	service, err := dockercloud.CreateService(opts)
	if err != nil {
		return err
	}

	d.SetId(service.Uuid)

	if err = service.Start(); err != nil {
		return fmt.Errorf("Unable to start service: %s", err)
	}

	stateConf := &resource.StateChangeConf{
		Pending:        []string{"Starting"},
		Target:         []string{"Running"},
		Refresh:        newServiceStateRefreshFunc(d, meta),
		Timeout:        60 * time.Minute,
		Delay:          10 * time.Second,
		MinTimeout:     3 * time.Second,
		NotFoundChecks: 60,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("Error waiting for service (%s) to become ready: %s", d.Id(), err)
	}

	return resourceDockercloudServiceRead(d, meta)
}

func resourceDockercloudServiceRead(d *schema.ResourceData, meta interface{}) error {
	service, err := dockercloud.GetService(d.Id())
	if err != nil {
		if err.(dockercloud.HttpError).StatusCode == 404 {
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
	d.Set("deployment_strategy", service.Deployment_strategy)
	d.Set("autorestart", service.Autorestart)
	d.Set("autodestroy", service.Autodestroy)
	d.Set("autoredeploy", service.Autoredeploy)
	d.Set("privileged", service.Privileged)
	d.Set("sequential_deployment", service.Sequential_deployment)
	d.Set("container_count", service.Target_num_containers)
	d.Set("roles", service.Roles)
	d.Set("tags", flattenTags(service.Tags))
	d.Set("bindings", flattenBindings(service.Bindings))
	d.Set("env", flattenEnv(service.Container_envvars))
	d.Set("links", flattenLinks(service.Linked_to_service))
	d.Set("ports", flattenPorts(service.Container_ports))
	d.Set("uri", service.Resource_uri)
	d.Set("public_dns", service.Public_dns)

	return nil
}

func resourceDockercloudServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	var change bool
	var opts dockercloud.ServiceCreateRequest

	if d.HasChange("image") {
		_, v := d.GetChange("image")
		opts.Image = v.(string)
		change = true
	}

	if d.HasChange("entrypoint") {
		_, v := d.GetChange("entrypoint")
		opts.Entrypoint = v.(string)
		change = true
	}

	if d.HasChange("command") {
		_, v := d.GetChange("command")
		opts.Run_command = v.(string)
		change = true
	}

	if d.HasChange("workdir") {
		_, v := d.GetChange("workdir")
		opts.Working_dir = v.(string)
		change = true
	}

	if d.HasChange("pid") {
		_, v := d.GetChange("pid")
		opts.Pid = v.(string)
		change = true
	}

	if d.HasChange("net") {
		_, v := d.GetChange("net")
		opts.Net = v.(string)
		change = true
	}

	if d.HasChange("deployment_strategy") {
		_, v := d.GetChange("deployment_strategy")
		opts.Deployment_strategy = v.(string)
		change = true
	}

	if d.HasChange("autorestart") {
		_, v := d.GetChange("autorestart")
		opts.Autorestart = v.(string)
		change = true
	}

	if d.HasChange("autodestroy") {
		_, v := d.GetChange("autodestroy")
		opts.Autodestroy = v.(string)
		change = true
	}

	if d.HasChange("autoredeploy") {
		_, v := d.GetChange("autoredeploy")
		opts.Autoredeploy = v.(bool)
		change = true
	}

	if d.HasChange("privileged") {
		_, v := d.GetChange("privileged")
		opts.Privileged = v.(bool)
		change = true
	}

	if d.HasChange("sequential_deployment") {
		_, v := d.GetChange("sequential_deployment")
		opts.Sequential_deployment = v.(bool)
		change = true
	}

	if d.HasChange("container_count") {
		_, v := d.GetChange("container_count")
		opts.Target_num_containers = v.(int)
	}

	if d.HasChange("roles") {
		_, v := d.GetChange("roles")
		opts.Roles = stringListToSlice(v.([]interface{}))
		change = true
	}

	if d.HasChange("tags") {
		_, v := d.GetChange("tags")
		opts.Tags = stringListToSlice(v.([]interface{}))
	}

	if d.HasChange("bindings") {
		_, v := d.GetChange("bindings")
		opts.Bindings = bindingsSetToServiceBinding(v.(*schema.Set))
		change = true
	}

	if d.HasChange("env") {
		_, v := d.GetChange("env")
		opts.Container_envvars = envSetToContainerEnvvar(v.(*schema.Set))
		change = true
	}

	if d.HasChange("links") {
		_, v := d.GetChange("links")
		opts.Linked_to_service = linksSetToServiceLinkInfo(v.(*schema.Set))
		change = true
	}

	if d.HasChange("ports") {
		_, v := d.GetChange("ports")
		opts.Container_ports = portsSetToContainerPortInfo(v.(*schema.Set))
		change = true
	}

	service, err := dockercloud.GetService(d.Id())
	if err != nil {
		return fmt.Errorf("Error retrieving service (%s): %s", d.Id(), err)
	}

	if err := service.Update(opts); err != nil {
		return fmt.Errorf("Error updating service: %s", err)
	}

	if d.Get("redeploy_on_change").(bool) && change {
		reuse := d.Get("reuse_existing_volumes").(bool)
		if err := service.Redeploy(dockercloud.ReuseVolumesOption{Reuse: reuse}); err != nil {
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

	return resourceDockercloudServiceRead(d, meta)
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

func bindingsSetToServiceBinding(s *schema.Set) []dockercloud.ServiceBinding {
	serviceBindings := make([]dockercloud.ServiceBinding, s.Len())

	for i, b := range s.List() {
		var serviceBinding dockercloud.ServiceBinding

		binding := b.(map[string]interface{})

		serviceBinding.Rewritable = binding["rewritable"].(bool)

		v, ok := binding["container_path"]
		if ok {
			serviceBinding.Container_path = v.(string)
		}

		v, ok = binding["host_path"]
		if ok {
			serviceBinding.Host_path = v.(string)
		}

		v, ok = binding["volumes_from"]
		if ok {
			serviceBinding.Volumes_from = v.(string)
		}

		serviceBindings[i] = serviceBinding
	}

	return serviceBindings
}

func flattenBindings(b []dockercloud.ServiceBinding) *schema.Set {
	bindings := make([]interface{}, len(b))

	for i, binding := range b {
		bindings[i] = map[string]interface{}{
			"container_path": binding.Container_path,
			"host_path":      binding.Host_path,
			"rewritable":     binding.Rewritable,
			"volumes_from":   binding.Volumes_from,
		}
	}

	return schema.NewSet(bindingsHash, bindings)
}

func bindingsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	if v, ok := m["container_path"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["host_path"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	if v, ok := m["rewritable"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(bool)))
	}

	if v, ok := m["volumes_from"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func envSetToContainerEnvvar(s *schema.Set) []dockercloud.ContainerEnvvar {
	containerEnvvars := make([]dockercloud.ContainerEnvvar, s.Len())

	for i, e := range s.List() {
		envvar := e.(map[string]interface{})

		containerEnvvars[i] = dockercloud.ContainerEnvvar{
			Key:   envvar["key"].(string),
			Value: envvar["value"].(string),
		}
	}

	return containerEnvvars
}

func flattenEnv(e []dockercloud.ContainerEnvvar) *schema.Set {
	envvars := make([]interface{}, len(e))

	for i, env := range e {
		envvars[i] = map[string]interface{}{
			"key":   env.Key,
			"value": env.Value,
		}
	}

	return schema.NewSet(envHash, envvars)
}

func envHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["key"].(string)))
	buf.WriteString(fmt.Sprintf("%s", m["value"].(string)))

	return hashcode.String(buf.String())
}

func linksSetToServiceLinkInfo(s *schema.Set) []dockercloud.ServiceLinkInfo {
	serviceLinks := make([]dockercloud.ServiceLinkInfo, s.Len())

	for i, l := range s.List() {
		var serviceLink dockercloud.ServiceLinkInfo

		link := l.(map[string]interface{})

		serviceLink.To_service = link["to"].(string)

		v, ok := link["name"]
		if ok {
			serviceLink.Name = v.(string)
		}

		serviceLinks[i] = serviceLink
	}

	return serviceLinks
}

func flattenLinks(l []dockercloud.ServiceLinkInfo) *schema.Set {
	links := make([]interface{}, len(l))

	for i, link := range l {
		links[i] = map[string]interface{}{
			"to":   link.To_service,
			"name": link.Name,
		}
	}

	return schema.NewSet(linksHash, links)
}

func linksHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%s-", m["to"].(string)))

	if v, ok := m["name"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func portsSetToContainerPortInfo(s *schema.Set) []dockercloud.ContainerPortInfo {
	containerPorts := make([]dockercloud.ContainerPortInfo, s.Len())

	for i, p := range s.List() {
		var containerPort dockercloud.ContainerPortInfo

		port := p.(map[string]interface{})

		containerPort.Inner_port = port["internal"].(int)
		containerPort.Protocol = port["protocol"].(string)

		v, ok := port["external"]
		if ok {
			containerPort.Outer_port = v.(int)
		}

		containerPorts[i] = containerPort
	}

	return containerPorts
}

func flattenPorts(p []dockercloud.ContainerPortInfo) *schema.Set {
	ports := make([]interface{}, len(p))

	for i, port := range p {
		ports[i] = map[string]interface{}{
			"internal": port.Inner_port,
			"external": port.Outer_port,
			"protocol": port.Protocol,
		}
	}

	return schema.NewSet(portsHash, ports)
}

func portsHash(v interface{}) int {
	var buf bytes.Buffer
	m := v.(map[string]interface{})

	buf.WriteString(fmt.Sprintf("%d-", m["internal"].(int)))

	if v, ok := m["external"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(int)))
	}

	if v, ok := m["protocol"]; ok {
		buf.WriteString(fmt.Sprintf("%v-", v.(string)))
	}

	return hashcode.String(buf.String())
}

func flattenTags(t []dockercloud.ServiceTag) []string {
	ret := make([]string, len(t))

	for i, tag := range t {
		ret[i] = tag.Name
	}

	return ret
}

func stringListToSlice(s []interface{}) []string {
	ret := make([]string, len(s))

	for i, v := range s {
		ret[i] = v.(string)
	}

	return ret
}

func stringSliceToList(s []string) []interface{} {
	ret := make([]interface{}, len(s))

	for i, v := range s {
		ret[i] = v
	}

	return ret
}
