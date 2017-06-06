package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
	api "k8s.io/kubernetes/pkg/apis/autoscaling/v1"
)

func expandHorizontalPodAutoscalerSpec(in []interface{}) api.HorizontalPodAutoscalerSpec {
	if len(in) == 0 || in[0] == nil {
		return api.HorizontalPodAutoscalerSpec{}
	}
	spec := api.HorizontalPodAutoscalerSpec{}
	m := in[0].(map[string]interface{})
	if v, ok := m["max_replicas"]; ok {
		spec.MaxReplicas = int32(v.(int))
	}
	if v, ok := m["min_replicas"].(int); ok && v > 0 {
		spec.MinReplicas = ptrToInt32(int32(v))
	}
	if v, ok := m["scale_target_ref"]; ok {
		spec.ScaleTargetRef = expandCrossVersionObjectReference(v.([]interface{}))
	}
	if v, ok := m["target_cpu_utilization_percentage"].(int); ok && v > 0 {
		spec.TargetCPUUtilizationPercentage = ptrToInt32(int32(v))
	}

	return spec
}

func expandCrossVersionObjectReference(in []interface{}) api.CrossVersionObjectReference {
	if len(in) == 0 || in[0] == nil {
		return api.CrossVersionObjectReference{}
	}
	ref := api.CrossVersionObjectReference{}
	m := in[0].(map[string]interface{})

	if v, ok := m["api_version"]; ok {
		ref.APIVersion = v.(string)
	}
	if v, ok := m["kind"]; ok {
		ref.Kind = v.(string)
	}
	if v, ok := m["name"]; ok {
		ref.Name = v.(string)
	}
	return ref
}

func flattenHorizontalPodAutoscalerSpec(spec api.HorizontalPodAutoscalerSpec) []interface{} {
	m := make(map[string]interface{}, 0)
	m["max_replicas"] = spec.MaxReplicas
	if spec.MinReplicas != nil {
		m["min_replicas"] = *spec.MinReplicas
	}
	m["scale_target_ref"] = flattenCrossVersionObjectReference(spec.ScaleTargetRef)
	if spec.TargetCPUUtilizationPercentage != nil {
		m["target_cpu_utilization_percentage"] = *spec.TargetCPUUtilizationPercentage
	}
	return []interface{}{m}
}

func flattenCrossVersionObjectReference(ref api.CrossVersionObjectReference) []interface{} {
	m := make(map[string]interface{}, 0)
	if ref.APIVersion != "" {
		m["api_version"] = ref.APIVersion
	}
	if ref.Kind != "" {
		m["kind"] = ref.Kind
	}
	if ref.Name != "" {
		m["name"] = ref.Name
	}
	return []interface{}{m}
}

func patchHorizontalPodAutoscalerSpec(prefix string, pathPrefix string, d *schema.ResourceData) []PatchOperation {
	ops := make([]PatchOperation, 0)

	if d.HasChange(prefix + "max_replicas") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/maxReplicas",
			Value: d.Get(prefix + "max_replicas").(int),
		})
	}
	if d.HasChange(prefix + "min_replicas") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/minReplicas",
			Value: d.Get(prefix + "min_replicas").(int),
		})
	}
	if d.HasChange(prefix + "scale_target_ref") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/scaleTargetRef",
			Value: expandCrossVersionObjectReference(d.Get(prefix + "scale_target_ref").([]interface{})),
		})
	}
	if d.HasChange(prefix + "target_cpu_utilization_percentage") {
		ops = append(ops, &ReplaceOperation{
			Path:  pathPrefix + "/targetCPUUtilizationPercentage",
			Value: d.Get(prefix + "target_cpu_utilization_percentage").(int),
		})
	}

	return ops
}
