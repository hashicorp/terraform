package kubernetes

import (
	"strconv"

	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/kubernetes/pkg/api/v1"
)

func flattenCapability(in []v1.Capability) []string {
	att := make([]string, 0, len(in))
	for i, v := range in {
		att[i] = string(v)
	}
	return att
}

func flattenContainerSecurityContext(in *v1.SecurityContext) []interface{} {
	att := make(map[string]interface{})

	if in.Privileged != nil {
		att["privileged"] = *in.Privileged
	}
	if in.ReadOnlyRootFilesystem != nil {
		att["read_only_root_filesystem"] = *in.ReadOnlyRootFilesystem
	}

	if in.RunAsNonRoot != nil {
		att["run_as_non_root"] = *in.RunAsNonRoot
	}
	if in.RunAsUser != nil {
		att["run_as_user"] = *in.RunAsUser
	}

	if in.SELinuxOptions != nil {
		att["se_linux_options"] = flattenSeLinuxOptions(in.SELinuxOptions)
	}
	if in.Capabilities != nil {
		att["capabilities"] = flattenSecurityCapabilities(in.Capabilities)
	}
	return []interface{}{att}

}

func flattenSecurityCapabilities(in *v1.Capabilities) []interface{} {
	att := make(map[string]interface{})

	if in.Add != nil {
		att["add"] = flattenCapability(in.Add)
	}
	if in.Drop != nil {
		att["drop"] = flattenCapability(in.Drop)
	}

	return []interface{}{att}
}

func flattenHandler(in *v1.Handler) []interface{} {
	att := make(map[string]interface{})

	if in.Exec != nil {
		att["exec"] = flattenExec(in.Exec)
	}
	if in.HTTPGet != nil {
		att["http_get"] = flattenHTTPGet(in.HTTPGet)
	}
	if in.TCPSocket != nil {
		att["tcp_socket"] = flattenTCPSocket(in.TCPSocket)
	}

	return []interface{}{att}
}

func flattenHTTPHeader(in []v1.HTTPHeader) []interface{} {
	att := make([]interface{}, len(in))
	for i, v := range in {
		m := map[string]interface{}{}

		if v.Name != "" {
			m["name"] = v.Name
		}

		if v.Value != "" {
			m["value"] = v.Value
		}
		att[i] = m
	}
	return att
}

func expandPort(v string) intstr.IntOrString {
	i, err := strconv.Atoi(v)
	if err != nil {
		return intstr.IntOrString{
			Type:   intstr.String,
			StrVal: v,
		}
	}
	return intstr.IntOrString{
		Type:   intstr.Int,
		IntVal: int32(i),
	}
}

func flattenHTTPGet(in *v1.HTTPGetAction) []interface{} {
	att := make(map[string]interface{})

	if in.Host != "" {
		att["host"] = in.Host
	}
	if in.Path != "" {
		att["path"] = in.Path
	}
	att["port"] = in.Port.String()
	att["scheme"] = in.Scheme
	if len(in.HTTPHeaders) > 0 {
		att["http_header"] = flattenHTTPHeader(in.HTTPHeaders)
	}

	return []interface{}{att}
}

func flattenTCPSocket(in *v1.TCPSocketAction) []interface{} {
	att := make(map[string]interface{})
	att["port"] = in.Port.String()
	return []interface{}{att}
}

func flattenExec(in *v1.ExecAction) []interface{} {
	att := make(map[string]interface{})
	if len(in.Command) > 0 {
		att["command"] = in.Command
	}
	return []interface{}{att}
}

func flattenLifeCycle(in *v1.Lifecycle) []interface{} {
	att := make(map[string]interface{})

	if in.PostStart != nil {
		att["post_start"] = flattenHandler(in.PostStart)
	}
	if in.PreStop != nil {
		att["pre_stop"] = flattenHandler(in.PreStop)
	}

	return []interface{}{att}
}

func flattenProbe(in *v1.Probe) []interface{} {
	att := make(map[string]interface{})

	att["failure_threshold"] = in.FailureThreshold
	att["initial_delay_seconds"] = in.InitialDelaySeconds
	att["period_seconds"] = in.PeriodSeconds
	att["success_threshold"] = in.SuccessThreshold
	att["timeout_seconds"] = in.TimeoutSeconds

	if in.Exec != nil {
		att["exec"] = flattenExec(in.Exec)
	}
	if in.HTTPGet != nil {
		att["http_get"] = flattenHTTPGet(in.HTTPGet)
	}
	if in.TCPSocket != nil {
		att["tcp_socket"] = flattenTCPSocket(in.TCPSocket)
	}

	return []interface{}{att}
}

func flattenConfigMapKeyRef(in *v1.ConfigMapKeySelector) []interface{} {
	att := make(map[string]interface{})

	if in.Key != "" {
		att["key"] = in.Key
	}
	if in.Name != "" {
		att["name"] = in.Name
	}
	return []interface{}{att}
}

func flattenObjectFieldSelector(in *v1.ObjectFieldSelector) []interface{} {
	att := make(map[string]interface{})

	if in.APIVersion != "" {
		att["api_version"] = in.APIVersion
	}
	if in.FieldPath != "" {
		att["field_path"] = in.FieldPath
	}
	return []interface{}{att}
}

func flattenResourceFieldSelector(in *v1.ResourceFieldSelector) []interface{} {
	att := make(map[string]interface{})

	if in.ContainerName != "" {
		att["container_name"] = in.ContainerName
	}
	if in.Resource != "" {
		att["resource"] = in.Resource
	}
	return []interface{}{att}
}

func flattenSecretKeyRef(in *v1.SecretKeySelector) []interface{} {
	att := make(map[string]interface{})

	if in.Key != "" {
		att["key"] = in.Key
	}
	if in.Name != "" {
		att["name"] = in.Name
	}
	return []interface{}{att}
}

func flattenValueFrom(in *v1.EnvVarSource) []interface{} {
	att := make(map[string]interface{})

	if in.ConfigMapKeyRef != nil {
		att["config_map_key_ref"] = flattenConfigMapKeyRef(in.ConfigMapKeyRef)
	}
	if in.ResourceFieldRef != nil {
		att["resource_field_ref"] = flattenResourceFieldSelector(in.ResourceFieldRef)
	}
	if in.SecretKeyRef != nil {
		att["secret_key_ref"] = flattenSecretKeyRef(in.SecretKeyRef)
	}
	if in.FieldRef != nil {
		att["field_ref"] = flattenObjectFieldSelector(in.FieldRef)
	}
	return []interface{}{att}
}

func flattenContainerVolumeMounts(in []v1.VolumeMount) ([]interface{}, error) {
	att := make([]interface{}, len(in))
	for i, v := range in {
		m := map[string]interface{}{}
		m["read_only"] = v.ReadOnly

		if v.MountPath != "" {
			m["mount_path"] = v.MountPath

		}
		if v.Name != "" {
			m["name"] = v.Name

		}
		if v.SubPath != "" {
			m["sub_path"] = v.SubPath
		}
		att[i] = m
	}
	return att, nil
}

func flattenContainerEnvs(in []v1.EnvVar) []interface{} {
	att := make([]interface{}, len(in))
	for i, v := range in {
		m := map[string]interface{}{}
		if v.Name != "" {
			m["name"] = v.Name
		}
		if v.Value != "" {
			m["value"] = v.Value
		}
		if v.ValueFrom != nil {
			m["value_from"] = flattenValueFrom(v.ValueFrom)
		}

		att[i] = m
	}
	return att
}

func flattenContainerPorts(in []v1.ContainerPort) []interface{} {
	att := make([]interface{}, len(in))
	for i, v := range in {
		m := map[string]interface{}{}
		m["container_port"] = v.ContainerPort
		if v.HostIP != "" {
			m["host_ip"] = v.HostIP
		}
		m["host_port"] = v.HostPort
		if v.Name != "" {
			m["name"] = v.Name
		}
		if v.Protocol != "" {
			m["protocol"] = v.Protocol
		}
		att[i] = m
	}
	return att
}

func flattenContainerResourceRequirements(in v1.ResourceRequirements) ([]interface{}, error) {
	att := make(map[string]interface{})
	if len(in.Limits) > 0 {
		att["limits"] = []interface{}{flattenResourceList(in.Limits)}
	}
	if len(in.Requests) > 0 {
		att["requests"] = []interface{}{flattenResourceList(in.Requests)}
	}
	return []interface{}{att}, nil
}

func flattenContainers(in []v1.Container) ([]interface{}, error) {
	att := make([]interface{}, len(in))
	for i, v := range in {
		c := make(map[string]interface{})
		c["image"] = v.Image
		c["name"] = v.Name
		if len(v.Command) > 0 {
			c["command"] = v.Command
		}
		if len(v.Args) > 0 {
			c["args"] = v.Args
		}

		c["image_pull_policy"] = v.ImagePullPolicy
		c["termination_message_path"] = v.TerminationMessagePath
		c["stdin"] = v.Stdin
		c["stdin_once"] = v.StdinOnce
		c["tty"] = v.TTY
		c["working_dir"] = v.WorkingDir
		res, err := flattenContainerResourceRequirements(v.Resources)
		if err != nil {
			return nil, err
		}

		c["resources"] = res
		if v.LivenessProbe != nil {
			c["liveness_probe"] = flattenProbe(v.LivenessProbe)
		}
		if v.ReadinessProbe != nil {
			c["readiness_probe"] = flattenProbe(v.ReadinessProbe)
		}
		if v.Lifecycle != nil {
			c["lifecycle"] = flattenLifeCycle(v.Lifecycle)
		}

		if v.SecurityContext != nil {
			c["security_context"] = flattenContainerSecurityContext(v.SecurityContext)
		}
		if len(v.Ports) > 0 {
			c["port"] = flattenContainerPorts(v.Ports)
		}
		if len(v.Env) > 0 {
			c["env"] = flattenContainerEnvs(v.Env)
		}

		if len(v.VolumeMounts) > 0 {
			volumeMounts, err := flattenContainerVolumeMounts(v.VolumeMounts)
			if err != nil {
				return nil, err
			}
			c["volume_mount"] = volumeMounts
		}
		att[i] = c
	}
	return att, nil
}

func expandContainers(ctrs []interface{}) ([]v1.Container, error) {
	if len(ctrs) == 0 {
		return []v1.Container{}, nil
	}
	cs := make([]v1.Container, len(ctrs))
	for i, c := range ctrs {
		ctr := c.(map[string]interface{})

		if image, ok := ctr["image"]; ok {
			cs[i].Image = image.(string)
		}
		if name, ok := ctr["name"]; ok {
			cs[i].Name = name.(string)
		}
		if command, ok := ctr["command"].([]interface{}); ok {
			cs[i].Command = expandStringSlice(command)
		}
		if args, ok := ctr["args"].([]interface{}); ok {
			cs[i].Args = expandStringSlice(args)
		}

		if v, ok := ctr["resources"].([]interface{}); ok && len(v) > 0 {

			var err error
			cs[i].Resources, err = expandContainerResourceRequirements(v)
			if err != nil {
				return cs, err
			}
		}

		if v, ok := ctr["port"].([]interface{}); ok && len(v) > 0 {
			var err error
			cs[i].Ports, err = expandContainerPort(v)
			if err != nil {
				return cs, err
			}
		}
		if v, ok := ctr["env"].([]interface{}); ok && len(v) > 0 {
			var err error
			cs[i].Env, err = expandContainerEnv(v)
			if err != nil {
				return cs, err
			}
		}

		if policy, ok := ctr["image_pull_policy"]; ok {
			cs[i].ImagePullPolicy = v1.PullPolicy(policy.(string))
		}

		if v, ok := ctr["lifecycle"].([]interface{}); ok && len(v) > 0 {
			cs[i].Lifecycle = expandLifeCycle(v)
		}

		if v, ok := ctr["liveness_probe"].([]interface{}); ok && len(v) > 0 {
			cs[i].LivenessProbe = expandProbe(v)
		}

		if v, ok := ctr["readiness_probe"].([]interface{}); ok && len(v) > 0 {
			cs[i].ReadinessProbe = expandProbe(v)
		}
		if v, ok := ctr["stdin"]; ok {
			cs[i].Stdin = v.(bool)
		}
		if v, ok := ctr["stdin_once"]; ok {
			cs[i].StdinOnce = v.(bool)
		}
		if v, ok := ctr["termination_message_path"]; ok {
			cs[i].TerminationMessagePath = v.(string)
		}
		if v, ok := ctr["tty"]; ok {
			cs[i].TTY = v.(bool)
		}
		if v, ok := ctr["security_context"].([]interface{}); ok && len(v) > 0 {
			cs[i].SecurityContext = expandContainerSecurityContext(v)
		}

		if v, ok := ctr["volume_mount"].([]interface{}); ok && len(v) > 0 {
			var err error
			cs[i].VolumeMounts, err = expandContainerVolumeMounts(v)
			if err != nil {
				return cs, err
			}
		}
	}
	return cs, nil
}

func expandExec(l []interface{}) *v1.ExecAction {
	if len(l) == 0 || l[0] == nil {
		return &v1.ExecAction{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.ExecAction{}
	if v, ok := in["command"].([]interface{}); ok && len(v) > 0 {
		obj.Command = expandStringSlice(v)
	}
	return &obj
}

func expandHTTPHeaders(l []interface{}) []v1.HTTPHeader {
	if len(l) == 0 {
		return []v1.HTTPHeader{}
	}
	headers := make([]v1.HTTPHeader, len(l))
	for i, c := range l {
		m := c.(map[string]interface{})
		if v, ok := m["name"]; ok {
			headers[i].Name = v.(string)
		}
		if v, ok := m["value"]; ok {
			headers[i].Value = v.(string)
		}
	}
	return headers
}
func expandContainerSecurityContext(l []interface{}) *v1.SecurityContext {
	if len(l) == 0 || l[0] == nil {
		return &v1.SecurityContext{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.SecurityContext{}
	if v, ok := in["privileged"]; ok {
		obj.Privileged = ptrToBool(v.(bool))
	}
	if v, ok := in["read_only_root_filesystem"]; ok {
		obj.ReadOnlyRootFilesystem = ptrToBool(v.(bool))
	}
	if v, ok := in["run_as_non_root"]; ok {
		obj.RunAsNonRoot = ptrToBool(v.(bool))
	}
	if v, ok := in["run_as_user"]; ok {
		obj.RunAsUser = ptrToInt64(int64(v.(int)))
	}
	if v, ok := in["se_linux_options"].([]interface{}); ok && len(v) > 0 {
		obj.SELinuxOptions = expandSeLinuxOptions(v)
	}
	if v, ok := in["capabilities"].([]interface{}); ok && len(v) > 0 {
		obj.Capabilities = expandSecurityCapabilities(v)
	}

	return &obj
}

func expandCapabilitySlice(s []interface{}) []v1.Capability {
	result := make([]v1.Capability, len(s), len(s))
	for k, v := range s {
		result[k] = v.(v1.Capability)
	}
	return result
}

func expandSecurityCapabilities(l []interface{}) *v1.Capabilities {
	if len(l) == 0 || l[0] == nil {
		return &v1.Capabilities{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.Capabilities{}
	if v, ok := in["add"].([]interface{}); ok {
		obj.Add = expandCapabilitySlice(v)
	}
	if v, ok := in["drop"].([]interface{}); ok {
		obj.Drop = expandCapabilitySlice(v)
	}
	return &obj
}

func expandTCPSocket(l []interface{}) *v1.TCPSocketAction {
	if len(l) == 0 || l[0] == nil {
		return &v1.TCPSocketAction{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.TCPSocketAction{}
	if v, ok := in["port"].(string); ok && len(v) > 0 {
		obj.Port = expandPort(v)
	}
	return &obj
}

func expandHTTPGet(l []interface{}) *v1.HTTPGetAction {
	if len(l) == 0 || l[0] == nil {
		return &v1.HTTPGetAction{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.HTTPGetAction{}
	if v, ok := in["host"].(string); ok && len(v) > 0 {
		obj.Host = v
	}
	if v, ok := in["path"].(string); ok && len(v) > 0 {
		obj.Path = v
	}
	if v, ok := in["scheme"].(string); ok && len(v) > 0 {
		obj.Scheme = v1.URIScheme(v)
	}

	if v, ok := in["port"].(string); ok && len(v) > 0 {
		obj.Port = expandPort(v)
	}

	if v, ok := in["http_header"].([]interface{}); ok && len(v) > 0 {
		obj.HTTPHeaders = expandHTTPHeaders(v)
	}
	return &obj
}

func expandProbe(l []interface{}) *v1.Probe {
	if len(l) == 0 || l[0] == nil {
		return &v1.Probe{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.Probe{}
	if v, ok := in["exec"].([]interface{}); ok && len(v) > 0 {
		obj.Exec = expandExec(v)
	}
	if v, ok := in["http_get"].([]interface{}); ok && len(v) > 0 {
		obj.HTTPGet = expandHTTPGet(v)
	}
	if v, ok := in["tcp_socket"].([]interface{}); ok && len(v) > 0 {
		obj.TCPSocket = expandTCPSocket(v)
	}
	if v, ok := in["failure_threshold"].(int); ok {
		obj.FailureThreshold = int32(v)
	}
	if v, ok := in["initial_delay_seconds"].(int); ok {
		obj.InitialDelaySeconds = int32(v)
	}
	if v, ok := in["period_seconds"].(int); ok {
		obj.PeriodSeconds = int32(v)
	}
	if v, ok := in["success_threshold"].(int); ok {
		obj.SuccessThreshold = int32(v)
	}
	if v, ok := in["timeout_seconds"].(int); ok {
		obj.TimeoutSeconds = int32(v)
	}

	return &obj
}

func expandHandlers(l []interface{}) *v1.Handler {
	if len(l) == 0 || l[0] == nil {
		return &v1.Handler{}
	}
	in := l[0].(map[string]interface{})
	obj := v1.Handler{}
	if v, ok := in["exec"].([]interface{}); ok && len(v) > 0 {
		obj.Exec = expandExec(v)
	}
	if v, ok := in["http_get"].([]interface{}); ok && len(v) > 0 {
		obj.HTTPGet = expandHTTPGet(v)
	}
	if v, ok := in["tcp_socket"].([]interface{}); ok && len(v) > 0 {
		obj.TCPSocket = expandTCPSocket(v)
	}
	return &obj

}
func expandLifeCycle(l []interface{}) *v1.Lifecycle {
	if len(l) == 0 || l[0] == nil {
		return &v1.Lifecycle{}
	}
	in := l[0].(map[string]interface{})
	obj := &v1.Lifecycle{}
	if v, ok := in["post_start"].([]interface{}); ok && len(v) > 0 {
		obj.PostStart = expandHandlers(v)
	}
	if v, ok := in["pre_stop"].([]interface{}); ok && len(v) > 0 {
		obj.PreStop = expandHandlers(v)
	}
	return obj
}

func expandContainerVolumeMounts(in []interface{}) ([]v1.VolumeMount, error) {
	if len(in) == 0 {
		return []v1.VolumeMount{}, nil
	}
	vmp := make([]v1.VolumeMount, len(in))
	for i, c := range in {
		p := c.(map[string]interface{})
		if mountPath, ok := p["mount_path"]; ok {
			vmp[i].MountPath = mountPath.(string)
		}
		if name, ok := p["name"]; ok {
			vmp[i].Name = name.(string)
		}
		if readOnly, ok := p["read_only"]; ok {
			vmp[i].ReadOnly = readOnly.(bool)
		}
		if subPath, ok := p["sub_path"]; ok {
			vmp[i].SubPath = subPath.(string)
		}
	}
	return vmp, nil
}

func expandContainerEnv(in []interface{}) ([]v1.EnvVar, error) {
	if len(in) == 0 {
		return []v1.EnvVar{}, nil
	}
	envs := make([]v1.EnvVar, len(in))
	for i, c := range in {
		p := c.(map[string]interface{})
		if name, ok := p["name"]; ok {
			envs[i].Name = name.(string)
		}
		if value, ok := p["value"]; ok {
			envs[i].Value = value.(string)
		}
		if v, ok := p["value_from"].([]interface{}); ok && len(v) > 0 {
			var err error
			envs[i].ValueFrom, err = expandEnvValueFrom(v)
			if err != nil {
				return envs, err
			}
		}
	}
	return envs, nil
}

func expandContainerPort(in []interface{}) ([]v1.ContainerPort, error) {
	if len(in) == 0 {
		return []v1.ContainerPort{}, nil
	}
	ports := make([]v1.ContainerPort, len(in))
	for i, c := range in {
		p := c.(map[string]interface{})
		if containerPort, ok := p["container_port"]; ok {
			ports[i].ContainerPort = int32(containerPort.(int))
		}
		if hostIP, ok := p["host_ip"]; ok {
			ports[i].HostIP = hostIP.(string)
		}
		if hostPort, ok := p["host_port"]; ok {
			ports[i].HostPort = int32(hostPort.(int))
		}
		if name, ok := p["name"]; ok {
			ports[i].Name = name.(string)
		}
		if protocol, ok := p["protocol"]; ok {
			ports[i].Protocol = v1.Protocol(protocol.(string))
		}
	}
	return ports, nil
}

func expandConfigMapKeyRef(r []interface{}) (*v1.ConfigMapKeySelector, error) {
	if len(r) == 0 || r[0] == nil {
		return &v1.ConfigMapKeySelector{}, nil
	}
	in := r[0].(map[string]interface{})
	obj := &v1.ConfigMapKeySelector{}

	if v, ok := in["key"].(string); ok {
		obj.Key = v
	}
	if v, ok := in["name"].(string); ok {
		obj.Name = v
	}
	return obj, nil

}
func expandFieldRef(r []interface{}) (*v1.ObjectFieldSelector, error) {
	if len(r) == 0 || r[0] == nil {
		return &v1.ObjectFieldSelector{}, nil
	}
	in := r[0].(map[string]interface{})
	obj := &v1.ObjectFieldSelector{}

	if v, ok := in["api_version"].(string); ok {
		obj.APIVersion = v
	}
	if v, ok := in["field_path"].(string); ok {
		obj.FieldPath = v
	}
	return obj, nil
}
func expandResourceFieldRef(r []interface{}) (*v1.ResourceFieldSelector, error) {
	if len(r) == 0 || r[0] == nil {
		return &v1.ResourceFieldSelector{}, nil
	}
	in := r[0].(map[string]interface{})
	obj := &v1.ResourceFieldSelector{}

	if v, ok := in["container_name"].(string); ok {
		obj.ContainerName = v
	}
	if v, ok := in["resource"].(string); ok {
		obj.Resource = v
	}
	return obj, nil
}
func expandSecretKeyRef(r []interface{}) (*v1.SecretKeySelector, error) {
	if len(r) == 0 || r[0] == nil {
		return &v1.SecretKeySelector{}, nil
	}
	in := r[0].(map[string]interface{})
	obj := &v1.SecretKeySelector{}

	if v, ok := in["key"].(string); ok {
		obj.Key = v
	}
	if v, ok := in["name"].(string); ok {
		obj.Name = v
	}
	return obj, nil
}

func expandEnvValueFrom(r []interface{}) (*v1.EnvVarSource, error) {
	if len(r) == 0 || r[0] == nil {
		return &v1.EnvVarSource{}, nil
	}
	in := r[0].(map[string]interface{})
	obj := &v1.EnvVarSource{}

	var err error
	if v, ok := in["config_map_key_ref"].([]interface{}); ok && len(v) > 0 {
		obj.ConfigMapKeyRef, err = expandConfigMapKeyRef(v)
		if err != nil {
			return obj, err
		}
	}
	if v, ok := in["field_ref"].([]interface{}); ok && len(v) > 0 {
		obj.FieldRef, err = expandFieldRef(v)
		if err != nil {
			return obj, err
		}
	}
	if v, ok := in["secret_key_ref"].([]interface{}); ok && len(v) > 0 {
		obj.SecretKeyRef, err = expandSecretKeyRef(v)
		if err != nil {
			return obj, err
		}
	}
	if v, ok := in["resource_field_ref"].([]interface{}); ok && len(v) > 0 {
		obj.ResourceFieldRef, err = expandResourceFieldRef(v)
		if err != nil {
			return obj, err
		}
	}
	return obj, nil

}

func expandContainerResourceRequirements(l []interface{}) (v1.ResourceRequirements, error) {
	if len(l) == 0 || l[0] == nil {
		return v1.ResourceRequirements{}, nil
	}
	in := l[0].(map[string]interface{})
	obj := v1.ResourceRequirements{}

	fn := func(in []interface{}) (v1.ResourceList, error) {
		for _, c := range in {
			p := c.(map[string]interface{})
			if p["cpu"] == "" {
				delete(p, "cpu")
			}
			if p["memory"] == "" {
				delete(p, "memory")
			}
			return expandMapToResourceList(p)
		}
		return nil, nil
	}

	var err error
	if v, ok := in["limits"].([]interface{}); ok && len(v) > 0 {
		obj.Limits, err = fn(v)
		if err != nil {
			return obj, err
		}
	}

	if v, ok := in["requests"].([]interface{}); ok && len(v) > 0 {
		obj.Requests, err = fn(v)
		if err != nil {
			return obj, err
		}
	}

	return obj, nil
}
