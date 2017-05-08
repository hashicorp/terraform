package kubernetes

import (
	"fmt"
	"net/url"
	"strings"

	"encoding/base64"
	"github.com/hashicorp/terraform/helper/schema"
	"k8s.io/kubernetes/pkg/api/resource"
	api "k8s.io/kubernetes/pkg/api/v1"
)

func idParts(id string) (string, string) {
	parts := strings.Split(id, "/")
	return parts[0], parts[1]
}

func buildId(meta api.ObjectMeta) string {
	return meta.Namespace + "/" + meta.Name
}

func expandMetadata(in []interface{}) api.ObjectMeta {
	meta := api.ObjectMeta{}
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

func flattenMetadata(meta api.ObjectMeta) []map[string]interface{} {
	m := make(map[string]interface{})
	m["annotations"] = filterAnnotations(meta.Annotations)
	if meta.GenerateName != "" {
		m["generate_name"] = meta.GenerateName
	}
	m["labels"] = meta.Labels
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

func filterAnnotations(m map[string]string) map[string]string {
	for k, _ := range m {
		if isInternalAnnotationKey(k) {
			delete(m, k)
		}
	}
	return m
}

func isInternalAnnotationKey(annotationKey string) bool {
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
