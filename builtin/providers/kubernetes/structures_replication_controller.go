package kubernetes

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/api/v1"
)

func flattenReplicationControllerSpec(in v1.ReplicationControllerSpec) ([]interface{}, error) {
	att := make(map[string]interface{})
	att["min_ready_seconds"] = in.MinReadySeconds

	if in.Replicas != nil {
		att["replicas"] = *in.Replicas
	}

	att["selector"] = in.Selector
	podSpec, err := flattenPodSpec(in.Template.Spec)
	if err != nil {
		return nil, err
	}
	att["template"] = podSpec

	return []interface{}{att}, nil
}

func expandReplicationControllerSpec(rc []interface{}) (v1.ReplicationControllerSpec, error) {
	obj := v1.ReplicationControllerSpec{}
	if len(rc) == 0 || rc[0] == nil {
		return obj, nil
	}
	in := rc[0].(map[string]interface{})
	obj.MinReadySeconds = int32(in["min_ready_seconds"].(int))
	obj.Replicas = ptrToInt32(int32(in["replicas"].(int)))
	obj.Selector = expandStringMap(in["selector"].(map[string]interface{}))
	podSpec, err := expandPodSpec(in["template"].([]interface{}))
	if err != nil {
		return obj, err
	}
	obj.Template = &v1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: obj.Selector,
		},
		Spec: podSpec,
	}

	return obj, nil
}
