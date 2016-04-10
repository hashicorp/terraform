package triton

import (
	"strings"
)

var (
	tritonTags = []string{
		"triton.cns.disable",
		"triton.cns.services",
		"triton.cns.reverse_ptr",
	}

	fromTritonMap = map[string]string{}
	toTritonMap   = map[string]string{}
)

func init() {
	for _, tritonTag := range tritonTags {
		// Transform: _ -> __ followed by . -> _
		// Inverse:   _ -> . followed by .. -> _
		terTag := strings.Replace(strings.Replace(tritonTag, "_", "__", -1),
			".", "_", -1)
		fromTritonMap[tritonTag] = terTag
		toTritonMap[terTag] = tritonTag
	}
}

// Returns Terraform's tags from Triton's machine tags
func tagsFromTritonTags(tritonTags map[string]string) map[string]string {
	tags := make(map[string]string)
	for k, v := range tritonTags {
		if new_k, ok := fromTritonMap[k]; ok {
			k = new_k
		}
		tags[k] = v
	}
	return tags
}

// Returns Triton's machine tags from Terraform tags
func tagsToTritonTags(tags map[string]interface{}) map[string]string {
	tritonTags := make(map[string]string)
	for k, v := range tags {
		if new_k, ok := toTritonMap[k]; ok {
			k = new_k
		}
		tritonTags[k] = v.(string)
	}
	return tritonTags
}
