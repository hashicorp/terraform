package kubernetes

import (
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/kubernetes/pkg/api"
)

func createStringList(_values []interface{}) []string, error {
	values := make(string, len(_values))
	for i, v := range _values {
		values[i] = v.(string)
	}
	return values, nil
}

func createContainers(_containers []interface{}) []api.Container, error {
	containers := make(api.Container, len(_containers))
	for i, v := range _containers {
		_container := v.(map[string]interface{})
		container := &api.Container{}

		container.Name = _container["name"]
		container.Image = _container["image"]

		if val, ok := _container["command"]; ok {
			if res, err := createStringList(val.([]interface{})); err == nil {
				container.Command = res
			} else {
				return nil, err
			}
		}

		if val, ok := _container["args"]; ok {
			if res, err := createStringList(val.([]interface{})); err == nil {
				container.Args = res
			} else {
				return nil, err
			}
		}

		if val, ok := _container["working_dir"]; ok {
			container.WorkingDir := val.(string)
		}

		if val, ok := _container["container_port"]; ok {
			if res, err := createContainerPorts(val.([]interface{})); err == nil {
				container.Ports = res
			} else {
				return nil, err
			}
		}

		if val, ok := _container["env"]; ok {
			if res, err := createEnvVars(val.([]interface{})); err == nil {
				container.Env = res
			} else {
				return nil, err
			}
		}

		if val, ok := _container["resources"]; ok {
			if res, err := createResourceRequirements(val.([]interface{})); err == nil {
				container.Resources = res
			} else {
				return nil, err
			}
		}

		if val, ok := _container["volume_mount"]; ok {
			if res, err := createVolumeMounts(val.([]interface{})); err == nil {
				container.VolumeMounts = res
			} else {
				return nil, err
			}
		}

		if val, ok := _container["liveness_probe"]; ok {
			if res, err := createProbe(val.([]interface{})); err == nil {
				container.LivenessProbe = res
			} else {
				return nil, err
			}
		}

		if val, ok := _container["readiness_probe"]; ok {
			if res, err := createProbe(val.([]interface{})); err == nil {
				container.ReadinessProbe = res
			} else {
				return nil, err
			}
		}

		if val, ok := _container["lifecycle"]; ok {
			if res, err := createLifecycle(val.([]interface{})); err == nil {
				container.Lifecycle = res
			} else {
				return nil, err
			}
		}

		container.TerminationMessagePath := _container["termination_message_path"].(string)

		container.ImagePullPolicy := _container["image_pull_policy"].(string)

		if val, ok := _container["security_context"]; ok {
			if res, err := createSecurityContext(val.([]interface{})); err == nil {
				container.SercurityContext = res
			} else {
				return nil, err
			}
		}

		if val, ok := _container["stdin"]; ok {
			container.Stdin = res.(bool)
		}

		if val, ok := _container["tty"]; ok {
			container.TTY = res.(bool)
		}

		containers[i] = container;
	}

	return containers, nil
}

func createContainerPorts(_ports []interface{}) []api.ContainerPort, error {
	ports := make([]api.ContainerPort, len(_ports))
	for i, v := range _ports {
		_port := v.(map[string]interface{})
		port := api.ContainerPort{}

		if val, ok := _port["name"]; ok {
			port.Name = val.(string)
		}

		if val, ok := _port["host_port"]; ok {
			port.HostPort = val.(int)
			if port.HostPort > 65536 || port.HostPort < 1 {
				return nil, fmt.Errorf("Error, `container.port.host_port` must be a " +
					"valid port number: 0 < port < 65536")
			}
		}

		if val, ok := _port["container_port"]; ok {
			port.HostPort = val.(int)
			if port.HostPort > 65536 || port.HostPort < 1 {
				return nil, fmt.Errorf("Error, `container.port.container_port` must be a " +
					"valid port number: 0 < port < 65536")
			}
		} else {
			return nil, fmt.Errorf("Error, `container_port` field of " +
				"`container.port` must be specified when `container.port` is.")
		}

		if val, ok := _port["protocol"]; ok {
			port.Protocol = val.(string)
			if port.Protocol != "UDP" && port.Protocol != "TCP" {
				return nil, fmt.Errorf("Error, `container.port.protocol` must " +
					"be either `UDP` or `TCP`")
			}
		} else {
			return nil, fmt.Errorf("Error, `protocol` field of " +
				"`container.port` must be specified when `container.port` is.")
		}

		if val, ok := _port["host_ip"]; ok {
			port.HostIP = val.(string)
		}

		ports[i] = port
	}

	return ports, nil
}

func createEnvVars(_env_vars []interface{}) []api.EnvVar, error {
	env_vars := make([]api.EnvVar, len(_env_vars))
	for i, v := range _env_vars {
		_env_var := v.(map[string]interface{})
		env_var := api.EnvVar{}

		if val, ok := _env_var["name"]; ok {
			env_var.Name = val.(string)
		} else {
			return nil, fmt.Errorf("Error, `container.env_var.name` field must " +
				"be specified when `container.env_var` is")
		}
	}
}
