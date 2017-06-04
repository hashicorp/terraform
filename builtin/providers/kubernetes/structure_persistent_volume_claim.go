package kubernetes

import (
	"github.com/hashicorp/terraform/helper/schema"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/pkg/api/v1"
)

// Flatteners

func flattenLabelSelector(in *metav1.LabelSelector) []interface{} {
	att := make(map[string]interface{})
	if len(in.MatchLabels) > 0 {
		att["match_labels"] = in.MatchLabels
	}
	if len(in.MatchExpressions) > 0 {
		att["match_expressions"] = flattenLabelSelectorRequirement(in.MatchExpressions)
	}
	return []interface{}{att}
}

func flattenLabelSelectorRequirement(in []metav1.LabelSelectorRequirement) []interface{} {
	att := make([]interface{}, len(in), len(in))
	for i, n := range in {
		m := make(map[string]interface{})
		m["key"] = n.Key
		m["operator"] = n.Operator
		m["values"] = newStringSet(schema.HashString, n.Values)
		att[i] = m
	}
	return att
}

func flattenPersistentVolumeClaimSpec(in v1.PersistentVolumeClaimSpec) []interface{} {
	att := make(map[string]interface{})
	att["access_modes"] = flattenPersistentVolumeAccessModes(in.AccessModes)
	att["resources"] = flattenResourceRequirements(in.Resources)
	if in.Selector != nil {
		att["selector"] = flattenLabelSelector(in.Selector)
	}
	if in.VolumeName != "" {
		att["volume_name"] = in.VolumeName
	}
	return []interface{}{att}
}

func flattenResourceRequirements(in v1.ResourceRequirements) []interface{} {
	att := make(map[string]interface{})
	if len(in.Limits) > 0 {
		att["limits"] = flattenResourceList(in.Limits)
	}
	if len(in.Requests) > 0 {
		att["requests"] = flattenResourceList(in.Requests)
	}
	return []interface{}{att}
}

// Expanders

func expandLabelSelector(l []interface{}) *metav1.LabelSelector {
	if len(l) == 0 || l[0] == nil {
		return &metav1.LabelSelector{}
	}
	in := l[0].(map[string]interface{})
	obj := &metav1.LabelSelector{}
	if v, ok := in["match_labels"].(map[string]interface{}); ok && len(v) > 0 {
		obj.MatchLabels = expandStringMap(v)
	}
	if v, ok := in["match_expressions"].([]interface{}); ok && len(v) > 0 {
		obj.MatchExpressions = expandLabelSelectorRequirement(v)
	}
	return obj
}

func expandLabelSelectorRequirement(l []interface{}) []metav1.LabelSelectorRequirement {
	if len(l) == 0 || l[0] == nil {
		return []metav1.LabelSelectorRequirement{}
	}
	obj := make([]metav1.LabelSelectorRequirement, len(l), len(l))
	for i, n := range l {
		in := n.(map[string]interface{})
		obj[i] = metav1.LabelSelectorRequirement{
			Key:      in["key"].(string),
			Operator: metav1.LabelSelectorOperator(in["operator"].(string)),
			Values:   sliceOfString(in["values"].(*schema.Set).List()),
		}
	}
	return obj
}

func expandPersistentVolumeClaimSpec(l []interface{}) (v1.PersistentVolumeClaimSpec, error) {
	if len(l) == 0 || l[0] == nil {
		return v1.PersistentVolumeClaimSpec{}, nil
	}
	in := l[0].(map[string]interface{})
	resourceRequirements, err := expandResourceRequirements(in["resources"].([]interface{}))
	if err != nil {
		return v1.PersistentVolumeClaimSpec{}, err
	}
	obj := v1.PersistentVolumeClaimSpec{
		AccessModes: expandPersistentVolumeAccessModes(in["access_modes"].(*schema.Set).List()),
		Resources:   resourceRequirements,
	}
	if v, ok := in["selector"].([]interface{}); ok && len(v) > 0 {
		obj.Selector = expandLabelSelector(v)
	}
	if v, ok := in["volume_name"].(string); ok {
		obj.VolumeName = v
	}
	return obj, nil
}

func expandResourceRequirements(l []interface{}) (v1.ResourceRequirements, error) {
	if len(l) == 0 || l[0] == nil {
		return v1.ResourceRequirements{}, nil
	}
	in := l[0].(map[string]interface{})
	obj := v1.ResourceRequirements{}
	if v, ok := in["limits"].(map[string]interface{}); ok && len(v) > 0 {
		var err error
		obj.Limits, err = expandMapToResourceList(v)
		if err != nil {
			return obj, err
		}
	}
	if v, ok := in["requests"].(map[string]interface{}); ok && len(v) > 0 {
		var err error
		obj.Requests, err = expandMapToResourceList(v)
		if err != nil {
			return obj, err
		}
	}
	return obj, nil
}
