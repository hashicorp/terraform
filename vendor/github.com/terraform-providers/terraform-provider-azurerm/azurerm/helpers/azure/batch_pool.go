package azure

import (
	"fmt"
	"log"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/services/batch/mgmt/2018-12-01/batch"
	"github.com/hashicorp/terraform/helper/schema"
)

// FlattenBatchPoolAutoScaleSettings flattens the auto scale settings for a Batch pool
func FlattenBatchPoolAutoScaleSettings(settings *batch.AutoScaleSettings) []interface{} {
	results := make([]interface{}, 0)

	if settings == nil {
		log.Printf("[DEBUG] settings is nil")
		return results
	}

	result := make(map[string]interface{})

	if settings.EvaluationInterval != nil {
		result["evaluation_interval"] = *settings.EvaluationInterval
	}

	if settings.Formula != nil {
		result["formula"] = *settings.Formula
	}

	return append(results, result)
}

// FlattenBatchPoolFixedScaleSettings flattens the fixed scale settings for a Batch pool
func FlattenBatchPoolFixedScaleSettings(settings *batch.FixedScaleSettings) []interface{} {
	results := make([]interface{}, 0)

	if settings == nil {
		log.Printf("[DEBUG] settings is nil")
		return results
	}

	result := make(map[string]interface{})

	if settings.TargetDedicatedNodes != nil {
		result["target_dedicated_nodes"] = *settings.TargetDedicatedNodes
	}

	if settings.TargetLowPriorityNodes != nil {
		result["target_low_priority_nodes"] = *settings.TargetLowPriorityNodes
	}

	if settings.ResizeTimeout != nil {
		result["resize_timeout"] = *settings.ResizeTimeout
	}

	return append(results, result)
}

// FlattenBatchPoolImageReference flattens the Batch pool image reference
func FlattenBatchPoolImageReference(image *batch.ImageReference) []interface{} {
	results := make([]interface{}, 0)
	if image == nil {
		log.Printf("[DEBUG] image is nil")
		return results
	}

	result := make(map[string]interface{})
	if image.Publisher != nil {
		result["publisher"] = *image.Publisher
	}
	if image.Offer != nil {
		result["offer"] = *image.Offer
	}
	if image.Sku != nil {
		result["sku"] = *image.Sku
	}
	if image.Version != nil {
		result["version"] = *image.Version
	}
	if image.ID != nil {
		result["id"] = *image.ID
	}

	return append(results, result)
}

// FlattenBatchPoolStartTask flattens a Batch pool start task
func FlattenBatchPoolStartTask(startTask *batch.StartTask) []interface{} {
	results := make([]interface{}, 0)

	if startTask == nil {
		log.Printf("[DEBUG] startTask is nil")
		return results
	}

	result := make(map[string]interface{})
	if startTask.CommandLine != nil {
		result["command_line"] = *startTask.CommandLine
	}
	if startTask.WaitForSuccess != nil {
		result["wait_for_success"] = *startTask.WaitForSuccess
	}
	if startTask.MaxTaskRetryCount != nil {
		result["max_task_retry_count"] = *startTask.MaxTaskRetryCount
	}

	if startTask.UserIdentity != nil {
		userIdentity := make(map[string]interface{})
		if startTask.UserIdentity.AutoUser != nil {
			autoUser := make(map[string]interface{})

			elevationLevel := string(startTask.UserIdentity.AutoUser.ElevationLevel)
			scope := string(startTask.UserIdentity.AutoUser.Scope)

			autoUser["elevation_level"] = elevationLevel
			autoUser["scope"] = scope

			userIdentity["auto_user"] = []interface{}{autoUser}
		} else {
			userIdentity["user_name"] = *startTask.UserIdentity.UserName
		}

		result["user_identity"] = []interface{}{userIdentity}
	}

	resourceFiles := make([]interface{}, 0)
	if startTask.ResourceFiles != nil {
		for _, armResourceFile := range *startTask.ResourceFiles {
			resourceFile := make(map[string]interface{})
			if armResourceFile.AutoStorageContainerName != nil {
				resourceFile["auto_storage_container_name"] = *armResourceFile.AutoStorageContainerName
			}
			if armResourceFile.StorageContainerURL != nil {
				resourceFile["storage_container_url"] = *armResourceFile.StorageContainerURL
			}
			if armResourceFile.HTTPURL != nil {
				resourceFile["http_url"] = *armResourceFile.HTTPURL
			}
			if armResourceFile.BlobPrefix != nil {
				resourceFile["blob_prefix"] = *armResourceFile.BlobPrefix
			}
			if armResourceFile.FilePath != nil {
				resourceFile["file_path"] = *armResourceFile.FilePath
			}
			if armResourceFile.FileMode != nil {
				resourceFile["file_mode"] = *armResourceFile.FileMode
			}
			resourceFiles = append(resourceFiles, resourceFile)
		}
	}

	if startTask.EnvironmentSettings != nil {
		environment := make(map[string]interface{})
		for _, envSetting := range *startTask.EnvironmentSettings {
			environment[*envSetting.Name] = *envSetting.Value
		}

		result["environment"] = environment
	}
	result["resource_file"] = resourceFiles

	return append(results, result)
}

// FlattenBatchPoolCertificateReferences flattens a Batch pool certificate reference
func FlattenBatchPoolCertificateReferences(armCertificates *[]batch.CertificateReference) []interface{} {
	if armCertificates == nil {
		return []interface{}{}
	}
	output := make([]interface{}, 0)

	for _, armCertificate := range *armCertificates {
		certificate := map[string]interface{}{}
		if armCertificate.ID != nil {
			certificate["id"] = *armCertificate.ID
		}
		certificate["store_location"] = string(armCertificate.StoreLocation)
		if armCertificate.StoreName != nil {
			certificate["store_name"] = *armCertificate.StoreName
		}
		visibility := &schema.Set{F: schema.HashString}
		if armCertificate.Visibility != nil {
			for _, armVisibility := range *armCertificate.Visibility {
				visibility.Add(string(armVisibility))
			}
		}
		certificate["visibility"] = visibility
		output = append(output, certificate)
	}
	return output
}

// FlattenBatchPoolContainerConfiguration flattens a Batch pool container configuration
func FlattenBatchPoolContainerConfiguration(armContainerConfiguration *batch.ContainerConfiguration) interface{} {

	result := make(map[string]interface{})

	if armContainerConfiguration == nil {
		return nil
	}

	if armContainerConfiguration.Type != nil {
		result["type"] = *armContainerConfiguration.Type
	}

	return []interface{}{result}
}

// ExpandBatchPoolImageReference expands Batch pool image reference
func ExpandBatchPoolImageReference(list []interface{}) (*batch.ImageReference, error) {
	if len(list) == 0 {
		return nil, fmt.Errorf("Error: storage image reference should be defined")
	}

	storageImageRef := list[0].(map[string]interface{})

	storageImageRefOffer := storageImageRef["offer"].(string)
	storageImageRefPublisher := storageImageRef["publisher"].(string)
	storageImageRefSku := storageImageRef["sku"].(string)
	storageImageRefVersion := storageImageRef["version"].(string)

	imageRef := &batch.ImageReference{
		Offer:     &storageImageRefOffer,
		Publisher: &storageImageRefPublisher,
		Sku:       &storageImageRefSku,
		Version:   &storageImageRefVersion,
	}

	return imageRef, nil
}

// ExpandBatchPoolContainerConfiguration expands the Batch pool container configuration
func ExpandBatchPoolContainerConfiguration(list []interface{}) (*batch.ContainerConfiguration, error) {
	if len(list) == 0 {
		return nil, nil
	}

	containerConfiguration := list[0].(map[string]interface{})
	containerType := containerConfiguration["type"].(string)

	containerConf := &batch.ContainerConfiguration{
		Type: &containerType,
	}

	return containerConf, nil
}

// ExpandBatchPoolCertificateReferences expands Batch pool certificate references
func ExpandBatchPoolCertificateReferences(list []interface{}) (*[]batch.CertificateReference, error) {
	result := []batch.CertificateReference{}

	for _, tempItem := range list {
		item := tempItem.(map[string]interface{})
		certificateReference, err := expandBatchPoolCertificateReference(item)
		if err != nil {
			return nil, err
		}
		result = append(result, *certificateReference)
	}
	return &result, nil
}

func expandBatchPoolCertificateReference(ref map[string]interface{}) (*batch.CertificateReference, error) {
	if len(ref) == 0 {
		return nil, fmt.Errorf("Error: storage image reference should be defined")
	}

	id := ref["id"].(string)
	storeLocation := ref["store_location"].(string)
	storeName := ref["store_name"].(string)
	visibilityRefs := ref["visibility"].(*schema.Set)
	visibility := []batch.CertificateVisibility{}
	if visibilityRefs != nil {
		for _, visibilityRef := range visibilityRefs.List() {
			visibility = append(visibility, batch.CertificateVisibility(visibilityRef.(string)))
		}
	}

	certificateReference := &batch.CertificateReference{
		ID:            &id,
		StoreLocation: batch.CertificateStoreLocation(storeLocation),
		StoreName:     &storeName,
		Visibility:    &visibility,
	}
	return certificateReference, nil
}

// ExpandBatchPoolStartTask expands Batch pool start task
func ExpandBatchPoolStartTask(list []interface{}) (*batch.StartTask, error) {
	if len(list) == 0 {
		return nil, fmt.Errorf("Error: batch pool start task should be defined")
	}

	startTaskValue := list[0].(map[string]interface{})

	startTaskCmdLine := startTaskValue["command_line"].(string)
	maxTaskRetryCount := int32(startTaskValue["max_task_retry_count"].(int))
	waitForSuccess := startTaskValue["wait_for_success"].(bool)

	userIdentityList := startTaskValue["user_identity"].([]interface{})
	if len(userIdentityList) == 0 {
		return nil, fmt.Errorf("Error: batch pool start task user identity should be defined")
	}

	userIdentityValue := userIdentityList[0].(map[string]interface{})
	userIdentity := batch.UserIdentity{}

	if autoUserValue, ok := userIdentityValue["auto_user"]; ok {
		autoUser := autoUserValue.([]interface{})
		if len(autoUser) != 0 {
			autoUserMap := autoUser[0].(map[string]interface{})
			userIdentity.AutoUser = &batch.AutoUserSpecification{
				ElevationLevel: batch.ElevationLevel(autoUserMap["elevation_level"].(string)),
				Scope:          batch.AutoUserScope(autoUserMap["scope"].(string)),
			}

		}
	} else if userNameValue, ok := userIdentityValue["username"]; ok {
		userName := userNameValue.(string)
		userIdentity.UserName = &userName
	} else {
		return nil, fmt.Errorf("Error: either auto_user or user_name should be speicfied for Batch pool start task")
	}

	resourceFileList := startTaskValue["resource_file"].([]interface{})
	resourceFiles := make([]batch.ResourceFile, 0)
	for _, resourceFileValueTemp := range resourceFileList {
		resourceFileValue := resourceFileValueTemp.(map[string]interface{})
		resourceFile := batch.ResourceFile{}
		if v, ok := resourceFileValue["auto_storage_container_name"]; ok {
			autoStorageContainerName := v.(string)
			if autoStorageContainerName != "" {
				resourceFile.AutoStorageContainerName = &autoStorageContainerName
			}
		}
		if v, ok := resourceFileValue["storage_container_url"]; ok {
			storageContainerURL := v.(string)
			if storageContainerURL != "" {
				resourceFile.StorageContainerURL = &storageContainerURL
			}
		}
		if v, ok := resourceFileValue["http_url"]; ok {
			httpURL := v.(string)
			if httpURL != "" {
				resourceFile.HTTPURL = &httpURL
			}
		}
		if v, ok := resourceFileValue["blob_prefix"]; ok {
			blobPrefix := v.(string)
			if blobPrefix != "" {
				resourceFile.BlobPrefix = &blobPrefix
			}
		}
		if v, ok := resourceFileValue["file_path"]; ok {
			filePath := v.(string)
			if filePath != "" {
				resourceFile.FilePath = &filePath
			}
		}
		if v, ok := resourceFileValue["file_mode"]; ok {
			fileMode := v.(string)
			if fileMode != "" {
				resourceFile.FileMode = &fileMode
			}
		}
		resourceFiles = append(resourceFiles, resourceFile)
	}

	startTask := &batch.StartTask{
		CommandLine:       &startTaskCmdLine,
		MaxTaskRetryCount: &maxTaskRetryCount,
		WaitForSuccess:    &waitForSuccess,
		UserIdentity:      &userIdentity,
		ResourceFiles:     &resourceFiles,
	}

	// populate environment settings, if defined
	if environment := startTaskValue["environment"]; environment != nil {
		envMap := environment.(map[string]interface{})
		envSettings := make([]batch.EnvironmentSetting, 0)

		for k, v := range envMap {
			theValue := v.(string)
			theKey := k
			envSetting := batch.EnvironmentSetting{
				Name:  &theKey,
				Value: &theValue,
			}

			envSettings = append(envSettings, envSetting)
		}

		startTask.EnvironmentSettings = &envSettings
	}

	return startTask, nil
}

// ValidateAzureRMBatchPoolName validates the name of a Batch pool
func ValidateAzureRMBatchPoolName(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)
	if !regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(value) {
		errors = append(errors, fmt.Errorf(
			"any combination of alphanumeric characters including hyphens and underscores are allowed in %q: %q", k, value))
	}

	if 1 > len(value) {
		errors = append(errors, fmt.Errorf("%q cannot be less than 1 character: %q", k, value))
	}

	if len(value) > 64 {
		errors = append(errors, fmt.Errorf("%q cannot be longer than 64 characters: %q %d", k, value, len(value)))
	}

	return warnings, errors
}
