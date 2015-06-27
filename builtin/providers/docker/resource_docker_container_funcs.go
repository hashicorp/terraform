package docker

import (
	"errors"
	"fmt"
	"strconv"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

func resourceDockerContainerCreate(d *schema.ResourceData, meta interface{}) error {
	var err error
	client := meta.(*dc.Client)

	var data Data
	if err := fetchLocalImages(&data, client); err != nil {
		return err
	}

	image := d.Get("image").(string)
	if _, ok := data.DockerImages[image]; !ok {
		if _, ok := data.DockerImages[image+":latest"]; !ok {
			return fmt.Errorf("Unable to find image %s", image)
		} else {
			image = image + ":latest"
		}
	}

	// The awesome, wonderful, splendiferous, sensical
	// Docker API now lets you specify a HostConfig in
	// CreateContainerOptions, but in my testing it still only
	// actually applies HostConfig options set in StartContainer.
	// How cool is that?
	createOpts := dc.CreateContainerOptions{
		Name: d.Get("name").(string),
		Config: &dc.Config{
			Image:      image,
			Hostname:   d.Get("hostname").(string),
			Domainname: d.Get("domainname").(string),
		},
	}

	if v, ok := d.GetOk("env"); ok {
		createOpts.Config.Env = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("command"); ok {
		createOpts.Config.Cmd = stringListToStringSlice(v.([]interface{}))
	}

	exposedPorts := map[dc.Port]struct{}{}
	portBindings := map[dc.Port][]dc.PortBinding{}

	if v, ok := d.GetOk("ports"); ok {
		exposedPorts, portBindings = portSetToDockerPorts(v.(*schema.Set))
	}
	if len(exposedPorts) != 0 {
		createOpts.Config.ExposedPorts = exposedPorts
	}

	volumes := map[string]struct{}{}
	binds := []string{}
	volumesFrom := []string{}

	if v, ok := d.GetOk("volumes"); ok {
		volumes, binds, volumesFrom, err = volumeSetToDockerVolumes(v.(*schema.Set))
		if err != nil {
			return fmt.Errorf("Unable to parse volumes: %s", err)
		}
	}
	if len(volumes) != 0 {
		createOpts.Config.Volumes = volumes
	}

	var retContainer *dc.Container
	if retContainer, err = client.CreateContainer(createOpts); err != nil {
		return fmt.Errorf("Unable to create container: %s", err)
	}
	if retContainer == nil {
		return fmt.Errorf("Returned container is nil")
	}

	d.SetId(retContainer.ID)

	hostConfig := &dc.HostConfig{
		Privileged:      d.Get("privileged").(bool),
		PublishAllPorts: d.Get("publish_all_ports").(bool),
	}

	if len(portBindings) != 0 {
		hostConfig.PortBindings = portBindings
	}

	if len(binds) != 0 {
		hostConfig.Binds = binds
	}
	if len(volumesFrom) != 0 {
		hostConfig.VolumesFrom = volumesFrom
	}

	if v, ok := d.GetOk("dns"); ok {
		hostConfig.DNS = stringSetToStringSlice(v.(*schema.Set))
	}

	if v, ok := d.GetOk("links"); ok {
		hostConfig.Links = stringSetToStringSlice(v.(*schema.Set))
	}

	if err := client.StartContainer(retContainer.ID, hostConfig); err != nil {
		return fmt.Errorf("Unable to start container: %s", err)
	}

	return resourceDockerContainerRead(d, meta)
}

func resourceDockerContainerRead(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)

	apiContainer, err := fetchDockerContainer(d.Id(), client)
	if err != nil {
		return err
	}

	if apiContainer == nil {
		// This container doesn't exist anymore
		d.SetId("")

		return nil
	}

	container, err := client.InspectContainer(apiContainer.ID)
	if err != nil {
		return fmt.Errorf("Error inspecting container %s: %s", apiContainer.ID, err)
	}

	if d.Get("must_run").(bool) && !container.State.Running {
		return resourceDockerContainerDelete(d, meta)
	}

	// Read Network Settings
	if container.NetworkSettings != nil {
		d.Set("ip_address", container.NetworkSettings.IPAddress)
		d.Set("ip_prefix_length", container.NetworkSettings.IPPrefixLen)
		d.Set("gateway", container.NetworkSettings.Gateway)
		d.Set("bridge", container.NetworkSettings.Bridge)
	}

	return nil
}

func resourceDockerContainerUpdate(d *schema.ResourceData, meta interface{}) error {
	return nil
}

func resourceDockerContainerDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*dc.Client)

	removeOpts := dc.RemoveContainerOptions{
		ID:            d.Id(),
		RemoveVolumes: true,
		Force:         true,
	}

	if err := client.RemoveContainer(removeOpts); err != nil {
		return fmt.Errorf("Error deleting container %s: %s", d.Id(), err)
	}

	d.SetId("")
	return nil
}

func stringListToStringSlice(stringList []interface{}) []string {
	ret := []string{}
	for _, v := range stringList {
		ret = append(ret, v.(string))
	}
	return ret
}

func stringSetToStringSlice(stringSet *schema.Set) []string {
	ret := []string{}
	if stringSet == nil {
		return ret
	}
	for _, envVal := range stringSet.List() {
		ret = append(ret, envVal.(string))
	}
	return ret
}

func fetchDockerContainer(ID string , client *dc.Client) (*dc.APIContainers, error) {
	apiContainers, err := client.ListContainers(dc.ListContainersOptions{All: true})

	if err != nil {
		return nil, fmt.Errorf("Error fetching container information from Docker: %s\n", err)
	}

	for _, apiContainer := range apiContainers {
		if apiContainer.ID == ID {
			return &apiContainer, nil
		}
	}

	return nil, nil
}

func portSetToDockerPorts(ports *schema.Set) (map[dc.Port]struct{}, map[dc.Port][]dc.PortBinding) {
	retExposedPorts := map[dc.Port]struct{}{}
	retPortBindings := map[dc.Port][]dc.PortBinding{}

	for _, portInt := range ports.List() {
		port := portInt.(map[string]interface{})
		internal := port["internal"].(int)
		protocol := port["protocol"].(string)

		exposedPort := dc.Port(strconv.Itoa(internal) + "/" + protocol)
		retExposedPorts[exposedPort] = struct{}{}

		external, extOk := port["external"].(int)
		ip, ipOk := port["ip"].(string)

		if extOk {
			portBinding := dc.PortBinding{
				HostPort: strconv.Itoa(external),
			}
			if ipOk {
				portBinding.HostIP = ip
			}
			retPortBindings[exposedPort] = append(retPortBindings[exposedPort], portBinding)
		}
	}

	return retExposedPorts, retPortBindings
}

func volumeSetToDockerVolumes(volumes *schema.Set) (map[string]struct{}, []string, []string, error) {
	retVolumeMap := map[string]struct{}{}
	retHostConfigBinds := []string{}
	retVolumeFromContainers := []string{}

	for _, volumeInt := range volumes.List() {
		volume := volumeInt.(map[string]interface{})
		fromContainer := volume["from_container"].(string)
		containerPath := volume["container_path"].(string)
		hostPath := volume["host_path"].(string)
		readOnly := volume["read_only"].(bool)

		switch {
		case len(fromContainer) == 0 && len(containerPath) == 0:
			return retVolumeMap, retHostConfigBinds, retVolumeFromContainers, errors.New("Volume entry without container path or source container")
		case len(fromContainer) != 0 && len(containerPath) != 0:
			return retVolumeMap, retHostConfigBinds, retVolumeFromContainers, errors.New("Both a container and a path specified in a volume entry")
		case len(fromContainer) != 0:
			retVolumeFromContainers = append(retVolumeFromContainers, fromContainer)
		case len(hostPath) != 0:
			readWrite := "rw"
			if readOnly {
				readWrite = "ro"
			}
			retVolumeMap[containerPath] = struct{}{}
			retHostConfigBinds = append(retHostConfigBinds, hostPath+":"+containerPath+":"+readWrite)
		default:
			retVolumeMap[containerPath] = struct{}{}
		}
	}

	return retVolumeMap, retHostConfigBinds, retVolumeFromContainers, nil
}
