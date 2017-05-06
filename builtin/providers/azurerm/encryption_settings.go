package azurerm

import (
	"github.com/Azure/azure-sdk-for-go/arm/compute"
	"github.com/Azure/azure-sdk-for-go/arm/disk"
	"github.com/hashicorp/terraform/helper/schema"
)

func encryptionSettingsSchema() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		MaxItems: 1,
		Elem: &schema.Resource{
			Schema: map[string]*schema.Schema{
				"enabled": {
					Type:     schema.TypeBool,
					Required: true,

					// Azure can change enabled from false to true, but not the other way around, so
					//   to keep idempotency, we'll conservatively set this to ForceNew=true
					// TODO: Is this the right behavior?
					ForceNew: true,
				},

				"disk_encryption_key": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"secret_url": {
								Type:     schema.TypeString,
								Required: true,
							},

							"source_vault_id": {
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
				"key_encryption_key": {
					Type:     schema.TypeList,
					Optional: true,
					MaxItems: 1,
					Elem: &schema.Resource{
						Schema: map[string]*schema.Schema{
							"key_url": {
								Type:     schema.TypeString,
								Required: true,
							},

							"source_vault_id": {
								Type:     schema.TypeString,
								Required: true,
							},
						},
					},
				},
			},
		},
	}
}

func flattenVmDiskEncryptionSettings(encryptionSettings *compute.DiskEncryptionSettings) map[string]interface{} {
	return map[string]interface{}{
		"enabled": *encryptionSettings.Enabled,
		"disk_encryption_key": []interface{}{
			map[string]interface{}{
				"secret_url":      *encryptionSettings.DiskEncryptionKey.SecretURL,
				"source_vault_id": *encryptionSettings.DiskEncryptionKey.SourceVault.ID,
			},
		},
		"key_encryption_key": []interface{}{
			map[string]interface{}{
				"key_url":         *encryptionSettings.KeyEncryptionKey.KeyURL,
				"source_vault_id": *encryptionSettings.KeyEncryptionKey.SourceVault.ID,
			},
		},
	}
}

func flattenManagedDiskEncryptionSettings(encryptionSettings *disk.EncryptionSettings) map[string]interface{} {
	return map[string]interface{}{
		"enabled": *encryptionSettings.Enabled,
		"disk_encryption_key": []interface{}{
			map[string]interface{}{
				"secret_url":      *encryptionSettings.DiskEncryptionKey.SecretURL,
				"source_vault_id": *encryptionSettings.DiskEncryptionKey.SourceVault.ID,
			},
		},
		"key_encryption_key": []interface{}{
			map[string]interface{}{
				"key_url":         *encryptionSettings.KeyEncryptionKey.KeyURL,
				"source_vault_id": *encryptionSettings.KeyEncryptionKey.SourceVault.ID,
			},
		},
	}
}

func expandVmDiskEncryptionSettings(settings map[string]interface{}) *compute.DiskEncryptionSettings {
	enabled := settings["enabled"].(bool)
	config := &compute.DiskEncryptionSettings{
		Enabled: &enabled,
	}

	if v := settings["disk_encryption_key"].([]interface{}); len(v) > 0 {
		dek := v[0].(map[string]interface{})

		secretURL := dek["secret_url"].(string)
		sourceVaultId := dek["source_vault_id"].(string)
		config.DiskEncryptionKey = &compute.KeyVaultSecretReference{
			SecretURL:   &secretURL,
			SourceVault: &compute.SubResource{ID: &sourceVaultId},
		}
	}

	if v := settings["key_encryption_key"].([]interface{}); len(v) > 0 {
		kek := v[0].(map[string]interface{})

		secretURL := kek["key_url"].(string)
		sourceVaultId := kek["source_vault_id"].(string)
		config.KeyEncryptionKey = &compute.KeyVaultKeyReference{
			KeyURL:      &secretURL,
			SourceVault: &compute.SubResource{ID: &sourceVaultId},
		}
	}

	return config
}

func expandManagedDiskEncryptionSettings(settings map[string]interface{}) *disk.EncryptionSettings {
	enabled := settings["enabled"].(bool)
	config := &disk.EncryptionSettings{
		Enabled: &enabled,
	}

	if v := settings["disk_encryption_key"].([]interface{}); len(v) > 0 {
		dek := v[0].(map[string]interface{})

		secretURL := dek["secret_url"].(string)
		sourceVaultId := dek["source_vault_id"].(string)
		config.DiskEncryptionKey = &disk.KeyVaultAndSecretReference{
			SecretURL:   &secretURL,
			SourceVault: &disk.SourceVault{ID: &sourceVaultId},
		}
	}

	if v := settings["key_encryption_key"].([]interface{}); len(v) > 0 {
		kek := v[0].(map[string]interface{})

		secretURL := kek["key_url"].(string)
		sourceVaultId := kek["source_vault_id"].(string)
		config.KeyEncryptionKey = &disk.KeyVaultAndKeyReference{
			KeyURL:      &secretURL,
			SourceVault: &disk.SourceVault{ID: &sourceVaultId},
		}
	}

	return config
}
