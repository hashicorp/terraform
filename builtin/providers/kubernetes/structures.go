package kubernetes

import (
	"fmt"
	"net/url"
	"strings"

	"encoding/base64"
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	api "k8s.io/kubernetes/pkg/api/v1"
)

func idParts(id string) (string, string) {
	parts := strings.Split(id, "/")
	return parts[0], parts[1]
}

func buildId(meta metav1.ObjectMeta) string {
	return meta.Namespace + "/" + meta.Name
}

func expandMetadata(in []interface{}) metav1.ObjectMeta {
	meta := metav1.ObjectMeta{}
	if len(in) < 1 {
		return meta
	}
	m := in[0].(map[string]interface{})

	meta.Annotations = expandStringMap(m["annotations"].(map[string]interface{}))
	meta.Labels = expandStringMap(m["labels"].(map[string]interface{}))

	if v, ok := m["generate_name"]; ok {
		meta.GenerateName = v.(string)
	}
	if v, ok := m["name"]; ok {
		meta.Name = v.(string)
	}
	if v, ok := m["namespace"]; ok {
		meta.Namespace = v.(string)
	}

	return meta
}

func patchMetadata(keyPrefix, pathPrefix string, d *schema.ResourceData) PatchOperations {
	ops := make([]PatchOperation, 0, 0)
	if d.HasChange(keyPrefix + "annotations") {
		oldV, newV := d.GetChange(keyPrefix + "annotations")
		diffOps := diffStringMap(pathPrefix+"annotations", oldV.(map[string]interface{}), newV.(map[string]interface{}))
		ops = append(ops, diffOps...)
	}
	if d.HasChange(keyPrefix + "labels") {
		oldV, newV := d.GetChange(keyPrefix + "labels")
		diffOps := diffStringMap(pathPrefix+"labels", oldV.(map[string]interface{}), newV.(map[string]interface{}))
		ops = append(ops, diffOps...)
	}
	return ops
}

func expandStringMap(m map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = v.(string)
	}
	return result
}

func expandStringSlice(s []interface{}) []string {
	result := make([]string, len(s), len(s))
	for k, v := range s {
		result[k] = v.(string)
	}
	return result
}

func flattenMetadata(meta metav1.ObjectMeta) []map[string]interface{} {
	m := make(map[string]interface{})
	m["annotations"] = removeInternalKeys(meta.Annotations)
	if meta.GenerateName != "" {
		m["generate_name"] = meta.GenerateName
	}
	m["labels"] = removeInternalKeys(meta.Labels)
	m["name"] = meta.Name
	m["resource_version"] = meta.ResourceVersion
	m["self_link"] = meta.SelfLink
	m["uid"] = fmt.Sprintf("%v", meta.UID)
	m["generation"] = meta.Generation

	if meta.Namespace != "" {
		m["namespace"] = meta.Namespace
	}

	return []map[string]interface{}{m}
}

func removeInternalKeys(m map[string]string) map[string]string {
	for k, _ := range m {
		if isInternalKey(k) {
			delete(m, k)
		}
	}
	return m
}

func isInternalKey(annotationKey string) bool {
	u, err := url.Parse("//" + annotationKey)
	if err == nil && strings.HasSuffix(u.Hostname(), "kubernetes.io") {
		return true
	}

	return false
}

func byteMapToStringMap(m map[string][]byte) map[string]string {
	result := make(map[string]string)
	for k, v := range m {
		result[k] = string(v)
	}
	return result
}

func ptrToString(s string) *string {
	return &s
}

func ptrToInt(i int) *int {
	return &i
}

func ptrToBool(b bool) *bool {
	return &b
}

func ptrToInt32(i int32) *int32 {
	return &i
}

func sliceOfString(slice []interface{}) []string {
	result := make([]string, len(slice), len(slice))
	for i, s := range slice {
		result[i] = s.(string)
	}
	return result
}

func base64EncodeStringMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range m {
		value := v.(string)
		result[k] = (base64.StdEncoding.EncodeToString([]byte(value)))
	}
	return result
}

func flattenResourceList(l api.ResourceList) map[string]string {
	m := make(map[string]string)
	for k, v := range l {
		m[string(k)] = v.String()
	}
	return m
}

func expandMapToResourceList(m map[string]interface{}) (api.ResourceList, error) {
	out := make(map[api.ResourceName]resource.Quantity)
	for stringKey, origValue := range m {
		key := api.ResourceName(stringKey)
		var value resource.Quantity

		if v, ok := origValue.(int); ok {
			q := resource.NewQuantity(int64(v), resource.DecimalExponent)
			value = *q
		} else if v, ok := origValue.(string); ok {
			var err error
			value, err = resource.ParseQuantity(v)
			if err != nil {
				return out, err
			}
		} else {
			return out, fmt.Errorf("Unexpected value type: %#v", origValue)
		}

		out[key] = value
	}
	return out, nil
}

func flattenPersistentVolumeAccessModes(in []api.PersistentVolumeAccessMode) *schema.Set {
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		out[i] = string(v)
	}
	return schema.NewSet(schema.HashString, out)
}

func expandPersistentVolumeAccessModes(s []interface{}) []api.PersistentVolumeAccessMode {
	out := make([]api.PersistentVolumeAccessMode, len(s), len(s))
	for i, v := range s {
		out[i] = api.PersistentVolumeAccessMode(v.(string))
	}
	return out
}

func flattenResourceQuotaSpec(in api.ResourceQuotaSpec) []interface{} {
	out := make([]interface{}, 1)

	m := make(map[string]interface{}, 0)
	m["hard"] = flattenResourceList(in.Hard)
	m["scopes"] = flattenResourceQuotaScopes(in.Scopes)

	out[0] = m
	return out
}

func expandResourceQuotaSpec(s []interface{}) (api.ResourceQuotaSpec, error) {
	out := api.ResourceQuotaSpec{}
	if len(s) < 1 {
		return out, nil
	}
	m := s[0].(map[string]interface{})

	if v, ok := m["hard"]; ok {
		list, err := expandMapToResourceList(v.(map[string]interface{}))
		if err != nil {
			return out, err
		}
		out.Hard = list
	}

	if v, ok := m["scopes"]; ok {
		out.Scopes = expandResourceQuotaScopes(v.(*schema.Set).List())
	}

	return out, nil
}

func flattenResourceQuotaScopes(in []api.ResourceQuotaScope) *schema.Set {
	out := make([]string, len(in), len(in))
	for i, scope := range in {
		out[i] = string(scope)
	}
	return newStringSet(schema.HashString, out)
}

func expandResourceQuotaScopes(s []interface{}) []api.ResourceQuotaScope {
	out := make([]api.ResourceQuotaScope, len(s), len(s))
	for i, scope := range s {
		out[i] = api.ResourceQuotaScope(scope.(string))
	}
	return out
}

func newStringSet(f schema.SchemaSetFunc, in []string) *schema.Set {
	var out = make([]interface{}, len(in), len(in))
	for i, v := range in {
		out[i] = v
	}
	return schema.NewSet(f, out)
}

func resourceListEquals(x, y api.ResourceList) bool {
	for k, v := range x {
		yValue, ok := y[k]
		if !ok {
			return false
		}
		if v.Cmp(yValue) != 0 {
			return false
		}
	}
	for k, v := range y {
		xValue, ok := x[k]
		if !ok {
			return false
		}
		if v.Cmp(xValue) != 0 {
			return false
		}
	}
	return true
}

func expandLimitRangeSpec(s []interface{}, isNew bool) (api.LimitRangeSpec, error) {
	out := api.LimitRangeSpec{}
	if len(s) < 1 || s[0] == nil {
		return out, nil
	}
	m := s[0].(map[string]interface{})

	if limits, ok := m["limit"].([]interface{}); ok {
		newLimits := make([]api.LimitRangeItem, len(limits), len(limits))

		for i, l := range limits {
			lrItem := api.LimitRangeItem{}
			limit := l.(map[string]interface{})

			if v, ok := limit["type"]; ok {
				lrItem.Type = api.LimitType(v.(string))
			}

			// defaultRequest is forbidden for Pod limits, even though it's set & returned by API
			// this is how we avoid sending it back
			if v, ok := limit["default_request"]; ok {
				drm := v.(map[string]interface{})
				if lrItem.Type == api.LimitTypePod && len(drm) > 0 {
					if isNew {
						return out, fmt.Errorf("limit.%d.default_request cannot be set for Pod limit", i)
					}
				} else {
					el, err := expandMapToResourceList(drm)
					if err != nil {
						return out, err
					}
					lrItem.DefaultRequest = el
				}
			}

			if v, ok := limit["default"]; ok {
				el, err := expandMapToResourceList(v.(map[string]interface{}))
				if err != nil {
					return out, err
				}
				lrItem.Default = el
			}
			if v, ok := limit["max"]; ok {
				el, err := expandMapToResourceList(v.(map[string]interface{}))
				if err != nil {
					return out, err
				}
				lrItem.Max = el
			}
			if v, ok := limit["max_limit_request_ratio"]; ok {
				el, err := expandMapToResourceList(v.(map[string]interface{}))
				if err != nil {
					return out, err
				}
				lrItem.MaxLimitRequestRatio = el
			}
			if v, ok := limit["min"]; ok {
				el, err := expandMapToResourceList(v.(map[string]interface{}))
				if err != nil {
					return out, err
				}
				lrItem.Min = el
			}

			newLimits[i] = lrItem
		}

		out.Limits = newLimits
	}

	return out, nil
}

func flattenLimitRangeSpec(in api.LimitRangeSpec) []interface{} {
	out := make([]interface{}, 1)
	limits := make([]interface{}, len(in.Limits), len(in.Limits))

	for i, l := range in.Limits {
		m := make(map[string]interface{}, 0)
		m["default"] = flattenResourceList(l.Default)
		m["default_request"] = flattenResourceList(l.DefaultRequest)
		m["max"] = flattenResourceList(l.Max)
		m["max_limit_request_ratio"] = flattenResourceList(l.MaxLimitRequestRatio)
		m["min"] = flattenResourceList(l.Min)
		m["type"] = string(l.Type)

		limits[i] = m
	}
	out[0] = map[string]interface{}{
		"limit": limits,
	}
	return out
}
