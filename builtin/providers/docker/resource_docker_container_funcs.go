package docker

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	dc "github.com/fsouza/go-dockerclient"
	"github.com/hashicorp/terraform/helper/schema"
)

var (
	creationTime time.Time
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
		}
		image = image + ":latest"
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

	if v, ok := d.GetOk("entrypoint"); ok {
		createOpts.Config.Entrypoint = stringListToStringSlice(v.([]interface{}))
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

	if v, ok := d.GetOk("labels"); ok {
		createOpts.Config.Labels = mapTypeMapValsToString(v.(map[string]interface{}))
	}

	hostConfig := &dc.HostConfig{
		Privileged:      d.Get("privileged").(bool),
		PublishAllPorts: d.Get("publish_all_ports").(bool),
		RestartPolicy: dc.RestartPolicy{
			Name:              d.Get("restart").(string),
			MaximumRetryCount: d.Get("max_retry_count").(int),
		},
		LogConfig: dc.LogConfig{
			Type: d.Get("log_driver").(string),
		},
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

	if v, ok := d.GetOk("memory"); ok {
		hostConfig.Memory = int64(v.(int)) * 1024 * 1024
	}

	if v, ok := d.GetOk("memory_swap"); ok {
		swap := int64(v.(int))
		if swap > 0 {
			swap = swap * 1024 * 1024
		}
		hostConfig.MemorySwap = swap
	}

	if v, ok := d.GetOk("cpu_shares"); ok {
		hostConfig.CPUShares = int64(v.(int))
	}

	if v, ok := d.GetOk("log_opts"); ok {
		hostConfig.LogConfig.Config = mapTypeMapValsToString(v.(map[string]interface{}))
	}

	if v, ok := d.GetOk("network_mode"); ok {
		hostConfig.NetworkMode = v.(string)
	}

	createOpts.HostConfig = hostConfig

	var retContainer *dc.Container
	if retContainer, err = client.CreateContainer(createOpts); err != nil {
		return fmt.Errorf("Unable to create container: %s", err)
	}
	if retContainer == nil {
		return fmt.Errorf("Returned container is nil")
	}

	d.SetId(retContainer.ID)

	creationTime = time.Now()
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

	var container *dc.Container

	loops := 1 // if it hasn't just been created, don't delay
	if !creationTime.IsZero() {
		loops = 30 // with 500ms spacing, 15 seconds; ought to be plenty
	}
	sleepTime := 500 * time.Millisecond

	for i := loops; i > 0; i-- {
		container, err = client.InspectContainer(apiContainer.ID)
		if err != nil {
			return fmt.Errorf("Error inspecting container %s: %s", apiContainer.ID, err)
		}

		if container.State.Running ||
			!container.State.Running && !d.Get("must_run").(bool) {
			break
		}

		if creationTime.IsZero() { // We didn't just create it, so don't wait around
			return resourceDockerContainerDelete(d, meta)
		}

		if container.State.FinishedAt.After(creationTime) {
			// It exited immediately, so error out so dependent containers
			// aren't started
			resourceDockerContainerDelete(d, meta)
			return fmt.Errorf("Container %s exited after creation, error was: %s", apiContainer.ID, container.State.Error)
		}

		time.Sleep(sleepTime)
	}

	// Handle the case of the for loop above running its course
	if !container.State.Running && d.Get("must_run").(bool) {
		resourceDockerContainerDelete(d, meta)
		return fmt.Errorf("Container %s failed to be in running state", apiContainer.ID)
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

func mapTypeMapValsToString(typeMap map[string]interface{}) map[string]string {
	mapped := make(map[string]string, len(typeMap))
	for k, v := range typeMap {
		mapped[k] = v.(string)
	}
	return mapped
}

func fetchDockerContainer(ID string, client *dc.Client) (*dc.APIContainers, error) {
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
