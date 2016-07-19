package ddcloud

import (
	"github.com/DimensionDataResearch/go-dd-cloud-compute/compute"
	"github.com/hashicorp/terraform/helper/schema"
	"log"
)

const (
	resourceKeyServerTag      = "tag"
	resourceKeyServerTagName  = "name"
	resourceKeyServerTagValue = "value"
)

func schemaServerTag() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeSet,
		Optional: true,
		Default:  nil,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				resourceKeyServerTagName: &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
				resourceKeyServerTagValue: &schema.Schema{
					Type:     schema.TypeString,
					Required: true,
				},
			},
		},
		Set: hashServerTag,
	}
}

// Apply configured tags to a server.
func applyServerTags(data *schema.ResourceData, apiClient *compute.Client) error {
	propertyHelper := propertyHelper(data)

	serverID := data.Id()

	log.Printf("Configuring tags for server '%s'...", serverID)

	configuredTags := propertyHelper.GetTags(resourceKeyServerTag)

	// TODO: Support multiple pages of results.
	serverTags, err := apiClient.GetAssetTags(serverID, compute.AssetTypeServer, nil)
	if err != nil {
		return err
	}

	// Capture any tags that are no-longer needed.
	unusedTags := &schema.Set{
		F: schema.HashString,
	}
	for _, tag := range serverTags.Items {
		unusedTags.Add(tag.Name)
	}
	for _, tag := range configuredTags {
		unusedTags.Remove(tag.Name)
	}

	log.Printf("Applying %d tags to server '%s'...", len(configuredTags), serverID)

	response, err := apiClient.ApplyAssetTags(serverID, compute.AssetTypeServer, configuredTags...)
	if err != nil {
		return err
	}

	if response.ResponseCode != compute.ResponseCodeOK {
		return response.ToError("Failed to apply %d tags to server '%s' (response code '%s'): %s", len(configuredTags), serverID, response.ResponseCode, response.Message)
	}

	// Trim unused tags (currently-configured tags will overwrite any existing values).
	if unusedTags.Len() > 0 {
		unusedTagNames := make([]string, unusedTags.Len())
		for index, unusedTagName := range unusedTags.List() {
			unusedTagNames[index] = unusedTagName.(string)
		}

		log.Printf("Removing %d unused tags from server '%s'...", len(unusedTagNames), serverID)

		response, err = apiClient.RemoveAssetTags(serverID, compute.AssetTypeServer, unusedTagNames...)
		if err != nil {
			return err
		}

		if response.ResponseCode != compute.ResponseCodeOK {
			return response.ToError("Failed to remove %d tags from server '%s' (response code '%s'): %s", len(configuredTags), serverID, response.ResponseCode, response.Message)
		}
	}

	return nil
}

// Read tags from a server and update resource data accordingly.
func readServerTags(data *schema.ResourceData, apiClient *compute.Client) error {
	propertyHelper := propertyHelper(data)

	serverID := data.Id()

	log.Printf("Reading tags for server '%s'...", serverID)

	result, err := apiClient.GetAssetTags(serverID, compute.AssetTypeServer, nil)
	if err != nil {
		return err
	}

	log.Printf("Read %d tags for server '%s'.", result.PageCount, serverID)

	// TODO: Handle multiple pages of results.

	tags := make([]compute.Tag, len(result.Items))
	for index, tagDetail := range result.Items {
		tags[index] = compute.Tag{
			Name:  tagDetail.Name,
			Value: tagDetail.Value,
		}
	}

	propertyHelper.SetTags(resourceKeyServerTag, tags)

	return nil
}

func hashServerTag(item interface{}) int {
	tagData := item.(map[string]interface{})

	return schema.HashString(
		tagData[resourceKeyServerTagName].(string),
	)
}
