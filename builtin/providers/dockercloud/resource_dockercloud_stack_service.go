package dockercloud

import (
	"fmt"
	"time"

	"github.com/docker/go-dockercloud/dockercloud"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockercloudStackService() *schema.Resource {
	s := serviceSchema()
	s["stack_uri"] = &schema.Schema{
		Type:     schema.TypeString,
		Required: true,
	}

	// The only part that differs from a service and a stack service
	// is the way it is created. The rest of the service functions can
	// be reused without modification.
	return &schema.Resource{
		Create: resourceDockercloudStackServiceCreate,
		Read:   resourceDockercloudServiceRead,
		Update: resourceDockercloudServiceUpdate,
		Delete: resourceDockercloudServiceDelete,
		Exists: resourceDockercloudServiceExists,
		Schema: s,
	}
}

func resourceDockercloudStackServiceCreate(d *schema.ResourceData, meta interface{}) error {
	stackURI := d.Get("stack_uri").(string)
	stack, err := dockercloud.GetStack(stackURI)
	if err != nil {
		return fmt.Errorf("Failed retrieving stack: %s", err)
	}

	serviceCreateRequest := newServiceCreateRequest(d)
	serviceCreateRequests := []dockercloud.ServiceCreateRequest{serviceCreateRequest}

	for _, s := range stack.Services {
		s, err := dockercloud.GetService(s)
		if err != nil {
			return fmt.Errorf("Failed retrieving stack services: %s", err)
		}
		serviceCreateRequests = append(serviceCreateRequests, serviceToServiceCreateRequest(s))
	}

	err = stack.Update(dockercloud.StackCreateRequest{
		Services: serviceCreateRequests,
	})
	if err != nil {
		return fmt.Errorf("Failed updating stack services: %s", err)
	}

	newStack, err := dockercloud.GetStack(stackURI)
	if err != nil {
		return fmt.Errorf("Failed retrieving stack: %s", err)
	}

	var serviceURI string
	for _, s := range newStack.Services {
		found := false
		for _, o := range stack.Services {
			if s == o {
				found = true
				break
			}
		}
		if !found {
			serviceURI = s
			break
		}
	}

	if serviceURI == "" {
		return fmt.Errorf("Could not get URI for service")
	}

	service, err := dockercloud.GetService(serviceURI)
	if err != nil {
		return fmt.Errorf("Failed retrieving service: %s", err)
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

func serviceToServiceCreateRequest(service dockercloud.Service) dockercloud.ServiceCreateRequest {
	return dockercloud.ServiceCreateRequest{
		Autodestroy:           service.Autodestroy,
		Autoredeploy:          service.Autoredeploy,
		Autorestart:           service.Autorestart,
		Bindings:              service.Bindings,
		Container_envvars:     service.Container_envvars,
		Container_ports:       service.Container_ports,
		Deployment_strategy:   service.Deployment_strategy,
		Entrypoint:            service.Entrypoint,
		Image:                 service.Image_name,
		Linked_to_service:     service.Linked_to_service,
		Name:                  service.Name,
		Net:                   service.Net,
		Pid:                   service.Pid,
		Privileged:            service.Privileged,
		Roles:                 service.Roles,
		Run_command:           service.Run_command,
		Sequential_deployment: service.Sequential_deployment,
		Tags: flattenTags(service.Tags),
		Target_num_containers: service.Target_num_containers,
		Working_dir:           service.Working_dir,
	}
}
