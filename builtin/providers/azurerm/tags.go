package azurerm

import (
	"errors"
	"fmt"

	"github.com/hashicorp/terraform/helper/schema"
)

func tagsSchema() *schema.Schema {
	return &schema.Schema{
		Type:         schema.TypeMap,
		Optional:     true,
		Computed:     true,
		ValidateFunc: validateAzureRMTags,
	}
}

func tagValueToString(v interface{}) (string, error) {
	switch value := v.(type) {
	case string:
		return value, nil
	case int:
		return fmt.Sprintf("%d", value), nil
	default:
		return "", fmt.Errorf("unknown tag type %T in tag value", value)
	}
}

func validateAzureRMTags(v interface{}, k string) (ws []string, es []error) {
	tagsMap := v.(map[string]interface{})

	if len(tagsMap) > 15 {
		es = append(es, errors.New("a maximum of 15 tags can be applied to each ARM resource"))
	}

	for k, v := range tagsMap {
		if len(k) > 512 {
			es = append(es, fmt.Errorf("the maximum length for a tag key is 512 characters: %q is %d characters", k, len(k)))
		}

		value, err := tagValueToString(v)
		if err != nil {
			es = append(es, err)
		} else if len(value) > 256 {
			es = append(es, fmt.Errorf("the maximum length for a tag value is 256 characters: the value for %q is %d characters", k, len(value)))
		}
	}

	return
}

func expandTags(tagsMap map[string]interface{}) *map[string]*string {
	output := make(map[string]*string, len(tagsMap))

	for i, v := range tagsMap {
		//Validate should have ignored this error already
		value, _ := tagValueToString(v)
		output[i] = &value
	}

	return &output
}

func flattenAndSetTags(d *schema.ResourceData, tagsMap *map[string]*string) {
	if tagsMap == nil {
		return
	}

	output := make(map[string]interface{}, len(*tagsMap))

	for i, v := range *tagsMap {
		output[i] = *v
	}

	d.Set("tags", output)
}
