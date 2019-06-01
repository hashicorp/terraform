package azure

import (
	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
)

// EncryptionClient provides methods to encrypt and decrypt data using Azure Key Vault
type EncryptionClient struct {
	KvClient              *keyvault.BaseClient
	kvInfo                *KeyVaultKeyInfo
	kvAlgorithmParameters *KeyVaultAlgorithmParameters
}

// KeyVaultKeyInfo stores information about Azure Key Vault service
type KeyVaultKeyInfo struct {
	vaultURL   string
	keyName    string
	keyVersion string
}

// KeyVaultAlgorithmParameters contains details about the public key size, algorithm to be used and encrypt/decrypt maximum block sizes
type KeyVaultAlgorithmParameters struct {
	keySizeBits           int
	encryptBlockSizeBytes int
	decryptBlockSizeBytes int
	kvAlgorithm           keyvault.JSONWebKeyEncryptionAlgorithm
}
