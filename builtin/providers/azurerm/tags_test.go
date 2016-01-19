package azurerm

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidateMaximumNumberOfARMTags(t *testing.T) {
	tagsMap := make(map[string]interface{})
	for i := 0; i < 16; i++ {
		tagsMap[fmt.Sprintf("key%d", i)] = fmt.Sprintf("value%d", i)
	}

	_, es := validateAzureRMTags(tagsMap, "tags")

	if len(es) != 1 {
		t.Fatal("Expected one validation error for too many tags")
	}

	if !strings.Contains(es[0].Error(), "a maximum of 15 tags") {
		t.Fatal("Wrong validation error message for too many tags")
	}
}

func TestValidateARMTagMaxKeyLength(t *testing.T) {
	tooLongKey := strings.Repeat("long", 128) + "a"
	tagsMap := make(map[string]interface{})
	tagsMap[tooLongKey] = "value"

	_, es := validateAzureRMTags(tagsMap, "tags")
	if len(es) != 1 {
		t.Fatal("Expected one validation error for a key which is > 512 chars")
	}

	if !strings.Contains(es[0].Error(), "maximum length for a tag key") {
		t.Fatal("Wrong validation error message maximum tag key length")
	}

	if !strings.Contains(es[0].Error(), tooLongKey) {
		t.Fatal("Expected validated error to contain the key name")
	}

	if !strings.Contains(es[0].Error(), "513") {
		t.Fatal("Expected the length in the validation error for tag key")
	}
}

func TestValidateARMTagMaxValueLength(t *testing.T) {
	tagsMap := make(map[string]interface{})
	tagsMap["toolong"] = strings.Repeat("long", 64) + "a"

	_, es := validateAzureRMTags(tagsMap, "tags")
	if len(es) != 1 {
		t.Fatal("Expected one validation error for a value which is > 256 chars")
	}

	if !strings.Contains(es[0].Error(), "maximum length for a tag value") {
		t.Fatal("Wrong validation error message for maximum tag value length")
	}

	if !strings.Contains(es[0].Error(), "toolong") {
		t.Fatal("Expected validated error to contain the key name")
	}

	if !strings.Contains(es[0].Error(), "257") {
		t.Fatal("Expected the length in the validation error for value")
	}
}

func TestExpandARMTags(t *testing.T) {
	testData := make(map[string]interface{})
	testData["key1"] = "value1"
	testData["key2"] = 21
	testData["key3"] = "value3"

	tempExpanded := expandTags(testData)
	expanded := *tempExpanded

	if len(expanded) != 3 {
		t.Fatalf("Expected 3 results in expanded tag map, got %d", len(expanded))
	}

	for k, v := range testData {
		var strVal string
		switch v.(type) {
		case string:
			strVal = v.(string)
		case int:
			strVal = fmt.Sprintf("%d", v.(int))
		}

		if *expanded[k] != strVal {
			t.Fatalf("Expanded value %q incorrect: expected %q, got %q", k, strVal, expanded[k])
		}
	}
}
