package kubernetes

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/resource"
	"k8s.io/kubernetes/pkg/util"
)

func createStringList(_values []interface{}) []string {
	values := make([]string, len(_values))
	for i, v := range _values {
		values[i] = v.(string)
	}
	return values
}

func createContainers(_containers []interface{}) []api.Container {
	containers := make([]api.Container, len(_containers))
	for i, v := range _containers {
		_container := v.(map[string]interface{})
		container := &api.Container{}

		container.Name = _container["name"].(string)
		container.Image = _container["image"].(string)

		if val, ok := _container["command"]; ok {
			container.Command = createStringList(val.([]interface{}))
		}

		if val, ok := _container["args"]; ok {
			container.Args = createStringList(val.([]interface{}))
		}

		if val, ok := _container["working_dir"]; ok {
			container.WorkingDir = val.(string)
		}

		if val, ok := _container["container_port"]; ok {
			container.Ports = createContainerPorts(val.([]interface{}))
		}

		if val, ok := _container["env"]; ok {
			container.Env = createEnvVars(val.([]interface{}))
		}

		if val, ok := _container["resources"]; ok {
			resources := createResourceRequirements(val.([]interface{}))
			if resources != nil {
				container.Resources = *resources
			}
		}

		if val, ok := _container["volume_mount"]; ok {
			container.VolumeMounts = createVolumeMounts(val.([]interface{}))
		}

		if val, ok := _container["liveness_probe"]; ok {
			container.LivenessProbe = createProbe(val.([]interface{}))
		}

		if val, ok := _container["readiness_probe"]; ok {
			container.ReadinessProbe = createProbe(val.([]interface{}))
		}

		if val, ok := _container["lifecycle"]; ok {
			container.Lifecycle = createLifecycle(val.([]interface{}))
		}

		container.TerminationMessagePath = _container["termination_message_path"].(string)

		container.ImagePullPolicy = api.PullPolicy(_container["image_pull_policy"].(string))

		if val, ok := _container["security_context"]; ok {
			container.SecurityContext = createSecurityContext(val.([]interface{}))
		}

		if val, ok := _container["stdin"]; ok {
			container.Stdin = val.(bool)
		}

		if val, ok := _container["tty"]; ok {
			container.TTY = val.(bool)
		}

		containers[i] = *container
	}

	return containers
}

func createContainerPorts(_ports []interface{}) []api.ContainerPort {
	ports := make([]api.ContainerPort, len(_ports))
	for i, v := range _ports {
		_port := v.(map[string]interface{})
		port := api.ContainerPort{}

		if val, ok := _port["name"]; ok {
			port.Name = val.(string)
		}

		if val, ok := _port["host_port"]; ok {
			port.HostPort = val.(int)
		}

		port.ContainerPort = _port["container_port"].(int)

		port.Protocol = api.Protocol(_port["Protocol"].(string))

		if val, ok := _port["host_ip"]; ok {
			port.HostIP = val.(string)
		}

		ports[i] = port
	}

	return ports
}

func createEnvVars(_env_vars []interface{}) []api.EnvVar {
	env_vars := make([]api.EnvVar, len(_env_vars))
	for i, v := range _env_vars {
		_env_var := v.(map[string]interface{})
		env_var := api.EnvVar{}

		env_var.Name = _env_var["name"].(string)

		if val, ok := _env_var["value"]; ok {
			env_var.Value = val.(string)
		}

		if val, ok := _env_var["value_from"]; ok {
			env_var.ValueFrom = createEnvVarSource(val.([]interface{}))
		}

		env_vars[i] = env_var
	}

	return env_vars
}

func createEnvVarSource(_env_var_sources []interface{}) *api.EnvVarSource {
	if len(_env_var_sources) == 0 {
		return nil
	} else {
		_env_var_source := _env_var_sources[0].(map[string]interface{})
		return &api.EnvVarSource{
			FieldRef: createObjectFieldSelector(_env_var_source["field_ref"].([]interface{})),
		}
	}
}

func createObjectFieldSelector(_field_refs []interface{}) *api.ObjectFieldSelector {
	if len(_field_refs) == 0 {
		return nil
	} else {
		_field_ref := _field_refs[0].(map[string]interface{})
		return &api.ObjectFieldSelector{
			APIVersion: _field_ref["api_version"].(string),
			FieldPath:  _field_ref["field_path"].(string),
		}
	}
}

func createResourceRequirements(_resource_reqs []interface{}) *api.ResourceRequirements {
	if len(_resource_reqs) == 0 {
		return nil
	} else {
		_resource_req := _resource_reqs[0].(map[string]interface{})
		resource_req := &api.ResourceRequirements{}
		if val, ok := _resource_req["limits"]; ok {
			resource_req.Limits = createResourceList(val.(map[string]interface{}))
		}

		if val, ok := _resource_req["requests"]; ok {
			resource_req.Requests = createResourceList(val.(map[string]interface{}))
		}

		return resource_req
	}
}

func createResourceList(_resource_list map[string]interface{}) map[api.ResourceName]resource.Quantity {
	resource_list := make(map[api.ResourceName]resource.Quantity, len(_resource_list))
	for k, v := range(_resource_list) {
		if q, err := resource.ParseQuantity(v.(string)); err == nil && q != nil {
			resource_list[api.ResourceName(k)] = *q
		}
	}
	return resource_list
}

func createVolumeMounts(_volume_mounts []interface{}) []api.VolumeMount {
	volume_mounts := make([]api.VolumeMount, len(_volume_mounts))
	for i, v := range _volume_mounts {
		_volume_mount := v.(map[string]interface{})
		volume_mount := api.VolumeMount{
			MountPath: _volume_mount["mount_path"].(string),
		}

		if val, ok := _volume_mount["name"]; ok {
			volume_mount.Name = val.(string)
		}

		if val, ok := _volume_mount["read_only"]; ok {
			volume_mount.ReadOnly = val.(bool)
		}

		volume_mounts[i] = volume_mount
	}

	return volume_mounts
}

func createProbe(_probes []interface{}) *api.Probe {
	if len(_probes) == 0 {
		return nil
	} else {
		_probe := _probes[0].(map[string]interface{})
		probe := &api.Probe{}
		if val, ok := _probe["handler"]; ok {
			handler := createHandler(val.([]interface{}))
			probe.Exec = handler.Exec
			probe.HTTPGet = handler.HTTPGet
			probe.TCPSocket = handler.TCPSocket
		}

		if val, ok := _probe["initial_delay_seconds"]; ok {
			probe.InitialDelaySeconds = int64(val.(int))
		}

		if val, ok := _probe["timeout_seconds"]; ok {
			probe.TimeoutSeconds = int64(val.(int))
		}

		return probe
	}
}

func createHandler(_handlers []interface{}) *api.Handler {
	if len(_handlers) == 0 {
		return nil
	} else {
		_handler := _handlers[0].(map[string]interface{})
		handler := &api.Handler{}
		if val, ok := _handler["exec"]; ok {
			handler.Exec = createExecAction(val.([]interface{}))
		}

		if val, ok := _handler["http_get"]; ok {
			handler.HTTPGet = createHttpGetAction(val.([]interface{}))
		}

		if val, ok := _handler["tcp_socket"]; ok {
			handler.TCPSocket = createTcpSocketAction(val.([]interface{}))
		}

		return handler
	}
}

func createExecAction(_execs []interface{}) *api.ExecAction {
	if len(_execs) == 0 {
		return nil
	} else {
		_exec := _execs[0].(map[string]interface{})
		exec := &api.ExecAction{}
		exec.Command = createStringList(_exec["command"].([]interface{}))
		return exec
	}
}

func createHttpGetAction(_http_gets []interface{}) *api.HTTPGetAction {
	if len(_http_gets) == 0 {
		return nil
	} else {
		_http_get := _http_gets[0].(map[string]interface{})
		httpGet := &api.HTTPGetAction{}

		httpGet.Port = util.NewIntOrStringFromInt(_http_get["port"].(int))

		if val, ok := _http_get["path"]; ok {
			httpGet.Path = val.(string)
		}

		if val, ok := _http_get["host"]; ok {
			httpGet.Host = val.(string)
		}

		if val, ok := _http_get["scheme"]; ok {
			httpGet.Scheme = api.URIScheme(val.(string))
		}

		return httpGet
	}
}

func createTcpSocketAction(_tcp_sockets []interface{}) *api.TCPSocketAction {
	if len(_tcp_sockets) == 0 {
		return nil
	} else {
		_tcp_socket := _tcp_sockets[0].(map[string]interface{})
		tcpSocket := &api.TCPSocketAction{}

		tcpSocket.Port = util.NewIntOrStringFromInt(_tcp_socket["port"].(int))

		return tcpSocket
	}
}

func createLifecycle(_lifecycles []interface{}) *api.Lifecycle {
	if len(_lifecycles) == 0 {
		return nil
	} else {
		_lifecycle := _lifecycles[0].(map[string]interface{})
		return &api.Lifecycle {
			PostStart: createHandler(_lifecycle["post_start"].([]interface{})),
			PreStop: createHandler(_lifecycle["pre_stop"].([]interface{})),
		}
	}
}

func createSecurityContext(_security_contexts []interface{}) *api.SecurityContext {
	if len(_security_contexts) == 0 {
		return nil
	} else {
		_security_context := _security_contexts[0].(map[string]interface{})
		securityContext := &api.SecurityContext{}

		if val, ok := _security_context["capabilities"]; ok {
			securityContext.Capabilities = createCapabilities(val.([]interface{}))
		}

		if val, ok := _security_context["privileged"]; ok {
			b := val.(bool)
			securityContext.Privileged = &b
		}

		if val, ok := _security_context["se_linux_options"]; ok {
			securityContext.SELinuxOptions = createSeLinuxOptions(val.([]interface{}))
		}

		if val, ok := _security_context["run_as_user"]; ok {
			v := int64(val.(int))
			securityContext.RunAsUser = &v
		}

		if val, ok := _security_context["run_as_non_root"]; ok {
			securityContext.RunAsNonRoot = val.(bool)
		}

		return securityContext
	}
}

func createCapabilities(_capabilities []interface{}) *api.Capabilities {
	if len(_capabilities) == 0 {
		return nil
	} else {
		_capability := _capabilities[0].(map[string]interface{})
		capability := &api.Capabilities{}

		if val, ok := _capability["add"]; ok {
			capability.Add = createCapabilityList(val.([]interface{}))
		}

		if val, ok := _capability["drop"]; ok {
			capability.Drop = createCapabilityList(val.([]interface{}))
		}

		return capability
	}
}

func createCapabilityList(_values []interface{}) []api.Capability {
	values := make([]api.Capability, len(_values))
	for i, v := range _values {
		values[i] = api.Capability(v.(string))
	}
	return values
}

func createSeLinuxOptions(_se_linux_options []interface{}) *api.SELinuxOptions {
	if len(_se_linux_options) == 0 {
		return nil
	} else {
		_se_linux_option := _se_linux_options[0].(map[string]interface{})
		seLinuxOption := &api.SELinuxOptions{}

		if val, ok := _se_linux_option["user"]; ok {
			seLinuxOption.User = val.(string)
		}

		if val, ok := _se_linux_option["role"]; ok {
			seLinuxOption.Role = val.(string)
		}

		if val, ok := _se_linux_option["type"]; ok {
			seLinuxOption.Type = val.(string)
		}

		if val, ok := _se_linux_option["level"]; ok {
			seLinuxOption.Level = val.(string)
		}

		return seLinuxOption
	}
}
