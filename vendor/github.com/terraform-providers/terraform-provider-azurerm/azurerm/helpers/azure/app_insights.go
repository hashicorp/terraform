package azure

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"
)

func ExpandApplicationInsightsAPIKeyLinkedProperties(v *schema.Set, appInsightsId string) *[]string {
	if v == nil {
		return &[]string{}
	}

	result := make([]string, v.Len())
	for i, prop := range v.List() {
		result[i] = fmt.Sprintf("%s/%s", appInsightsId, prop)
	}
	return &result
}

func FlattenApplicationInsightsAPIKeyLinkedProperties(props *[]string) *[]string {
	if props == nil {
		return &[]string{}
	}

	result := make([]string, len(*props))
	for i, prop := range *props {
		elems := strings.Split(prop, "/")
		result[i] = elems[len(elems)-1]
	}
	return &result
}
