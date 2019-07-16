package azure

import (
	"github.com/Azure/azure-sdk-for-go/services/keyvault/mgmt/2018-02-14/keyvault"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
	uuid "github.com/satori/go.uuid"
	"github.com/terraform-providers/terraform-provider-azurerm/azurerm/helpers/suppress"
)

func SchemaKeyVaultCertificatePermissions() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
			ValidateFunc: validation.StringInSlice([]string{
				string(keyvault.Backup),
				string(keyvault.Create),
				string(keyvault.Delete),
				string(keyvault.Deleteissuers),
				string(keyvault.Get),
				string(keyvault.Getissuers),
				string(keyvault.Import),
				string(keyvault.List),
				string(keyvault.Listissuers),
				string(keyvault.Managecontacts),
				string(keyvault.Manageissuers),
				string(keyvault.Purge),
				string(keyvault.Recover),
				string(keyvault.Restore),
				string(keyvault.Setissuers),
				string(keyvault.Update),
			}, true),
			DiffSuppressFunc: suppress.CaseDifference,
		},
	}
}

func SchemaKeyVaultKeyPermissions() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
			ValidateFunc: validation.StringInSlice([]string{
				string(keyvault.KeyPermissionsBackup),
				string(keyvault.KeyPermissionsCreate),
				string(keyvault.KeyPermissionsDecrypt),
				string(keyvault.KeyPermissionsDelete),
				string(keyvault.KeyPermissionsEncrypt),
				string(keyvault.KeyPermissionsGet),
				string(keyvault.KeyPermissionsImport),
				string(keyvault.KeyPermissionsList),
				string(keyvault.KeyPermissionsPurge),
				string(keyvault.KeyPermissionsRecover),
				string(keyvault.KeyPermissionsRestore),
				string(keyvault.KeyPermissionsSign),
				string(keyvault.KeyPermissionsUnwrapKey),
				string(keyvault.KeyPermissionsUpdate),
				string(keyvault.KeyPermissionsVerify),
				string(keyvault.KeyPermissionsWrapKey),
			}, true),
			DiffSuppressFunc: suppress.CaseDifference,
		},
	}
}

func SchemaKeyVaultSecretPermissions() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
			ValidateFunc: validation.StringInSlice([]string{
				string(keyvault.SecretPermissionsBackup),
				string(keyvault.SecretPermissionsDelete),
				string(keyvault.SecretPermissionsGet),
				string(keyvault.SecretPermissionsList),
				string(keyvault.SecretPermissionsPurge),
				string(keyvault.SecretPermissionsRecover),
				string(keyvault.SecretPermissionsRestore),
				string(keyvault.SecretPermissionsSet),
			}, true),
			DiffSuppressFunc: suppress.CaseDifference,
		},
	}
}

func SchemaKeyVaultStoragePermissions() *schema.Schema {
	return &schema.Schema{
		Type:     schema.TypeList,
		Optional: true,
		Elem: &schema.Schema{
			Type: schema.TypeString,
			ValidateFunc: validation.StringInSlice([]string{
				string(keyvault.StoragePermissionsBackup),
				string(keyvault.StoragePermissionsDelete),
				string(keyvault.StoragePermissionsDeletesas),
				string(keyvault.StoragePermissionsGet),
				string(keyvault.StoragePermissionsGetsas),
				string(keyvault.StoragePermissionsList),
				string(keyvault.StoragePermissionsListsas),
				string(keyvault.StoragePermissionsPurge),
				string(keyvault.StoragePermissionsRecover),
				string(keyvault.StoragePermissionsRegeneratekey),
				string(keyvault.StoragePermissionsRestore),
				string(keyvault.StoragePermissionsSet),
				string(keyvault.StoragePermissionsSetsas),
				string(keyvault.StoragePermissionsUpdate),
			}, false),
		},
	}
}

func ExpandKeyVaultAccessPolicies(input []interface{}) (*[]keyvault.AccessPolicyEntry, error) {
	output := make([]keyvault.AccessPolicyEntry, 0)

	for _, policySet := range input {
		policyRaw := policySet.(map[string]interface{})

		certificatePermissionsRaw := policyRaw["certificate_permissions"].([]interface{})
		keyPermissionsRaw := policyRaw["key_permissions"].([]interface{})
		secretPermissionsRaw := policyRaw["secret_permissions"].([]interface{})
		storagePermissionsRaw := policyRaw["storage_permissions"].([]interface{})

		policy := keyvault.AccessPolicyEntry{
			Permissions: &keyvault.Permissions{
				Certificates: ExpandCertificatePermissions(certificatePermissionsRaw),
				Keys:         ExpandKeyPermissions(keyPermissionsRaw),
				Secrets:      ExpandSecretPermissions(secretPermissionsRaw),
				Storage:      ExpandStoragePermissions(storagePermissionsRaw),
			},
		}

		tenantUUID := uuid.FromStringOrNil(policyRaw["tenant_id"].(string))
		policy.TenantID = &tenantUUID
		objectUUID := policyRaw["object_id"].(string)
		policy.ObjectID = &objectUUID

		if v := policyRaw["application_id"]; v != "" {
			applicationUUID := uuid.FromStringOrNil(v.(string))
			policy.ApplicationID = &applicationUUID
		}

		output = append(output, policy)
	}

	return &output, nil
}

func FlattenKeyVaultAccessPolicies(policies *[]keyvault.AccessPolicyEntry) []map[string]interface{} {
	result := make([]map[string]interface{}, 0)

	if policies == nil {
		return result
	}

	for _, policy := range *policies {
		policyRaw := make(map[string]interface{})

		if tenantId := policy.TenantID; tenantId != nil {
			policyRaw["tenant_id"] = tenantId.String()
		}

		if objectId := policy.ObjectID; objectId != nil {
			policyRaw["object_id"] = *objectId
		}

		if appId := policy.ApplicationID; appId != nil {
			policyRaw["application_id"] = appId.String()
		}

		if permissions := policy.Permissions; permissions != nil {
			certs := FlattenCertificatePermissions(permissions.Certificates)
			policyRaw["certificate_permissions"] = certs

			keys := FlattenKeyPermissions(permissions.Keys)
			policyRaw["key_permissions"] = keys

			secrets := FlattenSecretPermissions(permissions.Secrets)
			policyRaw["secret_permissions"] = secrets

			storage := FlattenStoragePermissions(permissions.Storage)
			policyRaw["storage_permissions"] = storage
		}

		result = append(result, policyRaw)
	}

	return result
}

func ExpandCertificatePermissions(input []interface{}) *[]keyvault.CertificatePermissions {
	output := make([]keyvault.CertificatePermissions, 0)

	for _, permission := range input {
		output = append(output, keyvault.CertificatePermissions(permission.(string)))
	}

	return &output
}

func FlattenCertificatePermissions(input *[]keyvault.CertificatePermissions) []interface{} {
	output := make([]interface{}, 0)

	if input != nil {
		for _, certificatePermission := range *input {
			output = append(output, string(certificatePermission))
		}
	}

	return output
}

func ExpandKeyPermissions(keyPermissionsRaw []interface{}) *[]keyvault.KeyPermissions {
	output := make([]keyvault.KeyPermissions, 0)

	for _, permission := range keyPermissionsRaw {
		output = append(output, keyvault.KeyPermissions(permission.(string)))
	}
	return &output
}

func FlattenKeyPermissions(input *[]keyvault.KeyPermissions) []interface{} {
	output := make([]interface{}, 0)

	if input != nil {
		for _, keyPermission := range *input {
			output = append(output, string(keyPermission))
		}
	}

	return output
}

func ExpandSecretPermissions(input []interface{}) *[]keyvault.SecretPermissions {
	output := make([]keyvault.SecretPermissions, 0)

	for _, permission := range input {
		output = append(output, keyvault.SecretPermissions(permission.(string)))
	}

	return &output
}

func FlattenSecretPermissions(input *[]keyvault.SecretPermissions) []interface{} {
	output := make([]interface{}, 0)

	if input != nil {
		for _, secretPermission := range *input {
			output = append(output, string(secretPermission))
		}
	}

	return output
}

func ExpandStoragePermissions(input []interface{}) *[]keyvault.StoragePermissions {
	output := make([]keyvault.StoragePermissions, 0)

	for _, permission := range input {
		output = append(output, keyvault.StoragePermissions(permission.(string)))
	}

	return &output
}

func FlattenStoragePermissions(input *[]keyvault.StoragePermissions) []interface{} {
	output := make([]interface{}, 0)

	if input != nil {
		for _, storagePermission := range *input {
			output = append(output, string(storagePermission))
		}
	}

	return output
}
