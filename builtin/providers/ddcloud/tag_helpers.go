package ddcloud

import (
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
)

// Helper functions for working with tags

// Get all tag keys
// Returns a map of tag keys, keyed by name.
func getAllTagKeys(apiClient *compute.Client, tagName string) (allTagKeys map[string]compute.TagKey, err error) {
	allTagKeys = make(map[string]compute.TagKey)

	paging := compute.PagingInfo{
		PageNumber: 1,
		PageSize:   10,
	}
	haveTagKeys := true
	for haveTagKeys {
		var tagKeys *compute.TagKeys
		tagKeys, err = apiClient.ListTagKeys(paging.PageNumber, paging.PageSize)
		if err != nil {
			return
		}

		haveTagKeys = len(tagKeys.Items) > 0

		for _, tagKey := range tagKeys.Items {
			allTagKeys[tagKey.Name] = tagKey
		}
	}

	return
}
