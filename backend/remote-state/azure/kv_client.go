package azure

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
)

// BlobEncryptedState is the object stored on Azure blob storage
type BlobEncryptedState struct {
	KeyVaultKey    *KeyVaultKey   `json:"key_vault_key"`
	EncryptionKey  *EncryptionKey `json:"encryption_key"`
	EncryptedState []byte         `json:"encrypted_state"`
}

// KeyVaultKey stores information about the Azure Key Vault asymmetric key
type KeyVaultKey struct {
	VaultURL   string `json:"vault_url"`
	KeyName    string `json:"key_name"`
	KeyVersion string `json:"key_version"`
}

// EncryptionKey stores information about the symmetric encryption key
type EncryptionKey struct {
	Algorithm string `json:"algorithm"`
	Mode      string `json:"mode"`
	Key       string `json:"key"`
	Nonce     []byte `json:"nonce"`
}

// EncryptionClient provides methods to encrypt and decrypt data
type EncryptionClient struct {
	KvClient *keyvault.BaseClient
	KvKey    *KeyVaultKey
}

// NewEncryptionClient creates an EncryptionClient from a Key Vault key identifier and a Key Vault client
func NewEncryptionClient(keyVaultKeyIdentifier string, kvClient *keyvault.BaseClient) (*EncryptionClient, error) {
	if kvClient == nil {
		return nil, fmt.Errorf("kvClient cannot be nil")
	}

	kvKey, err := parseKeyVaultKey(keyVaultKeyIdentifier)
	if err != nil {
		return nil, err
	}

	return &EncryptionClient{KvClient: kvClient, KvKey: kvKey}, nil
}

func parseKeyVaultKey(keyVaultKeyIdentifier string) (*KeyVaultKey, error) {
	r, _ := regexp.Compile("https?://(.+)\\.vault\\.azure\\.net/keys/([^\\/.]+)/?([^\\/.]*)")

	str := r.FindStringSubmatch(keyVaultKeyIdentifier)
	if len(str) < 4 {
		return &KeyVaultKey{}, fmt.Errorf("Expected a key identifier from Key Vault. e.g.: https://keyvaultname.vault.azure.net/keys/myKey/99d67321dd9841af859129cd5551a871")
	}

	key := KeyVaultKey{}
	key.VaultURL = fmt.Sprintf("https://%s.vault.azure.net", str[1])
	key.KeyName = str[2]
	key.KeyVersion = str[3]

	return &key, nil
}

// Encrypt encrypts the input using AES-256/GCM and wraps the encryption key using an Azure Key Vault asymmetric key with RSA 1.5
func (e *EncryptionClient) Encrypt(ctx context.Context, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return make([]byte, 0), nil
	}

	encrypted, key, nonce, err := encrypt(data)
	if err != nil {
		return nil, err
	}

	wrappedKey, err := e.wrapKey(ctx, key)
	if err != nil {
		return nil, err
	}

	blob := &BlobEncryptedState{
		KeyVaultKey: e.KvKey,
		EncryptionKey: &EncryptionKey{
			Algorithm: "AES-256",
			Mode:      "GCM",
			Key:       *wrappedKey,
			Nonce:     nonce,
		},
		EncryptedState: encrypted,
	}

	b, err := json.Marshal(blob)
	if err != nil {
		return nil, err
	}
	return b, nil
}

// Decrypt unwraps the encryption key using an Azure Key Vault asymmetric key with RSA 1.5 and decrypts the input using AES-256/GCM
func (e *EncryptionClient) Decrypt(ctx context.Context, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return make([]byte, 0), nil
	}

	var blob BlobEncryptedState
	err := json.Unmarshal(data, &blob)
	if err != nil {
		return nil, err
	}

	key, err := e.unwrapKey(ctx, &blob.EncryptionKey.Key)
	if err != nil {
		return nil, err
	}

	decrypted, err := decrypt(blob.EncryptedState, key, blob.EncryptionKey.Nonce)
	if err != nil {
		return nil, err
	}

	return decrypted, nil
}

func (e *EncryptionClient) wrapKey(ctx context.Context, key []byte) (*string, error) {
	if err := e.validateKeyVaultKey(ctx); err != nil {
		return nil, err
	}

	keyEncoded := base64.RawURLEncoding.EncodeToString(key)

	parameters := keyvault.KeyOperationsParameters{
		Algorithm: keyvault.RSA15,
		Value:     &keyEncoded,
	}

	result, err := e.KvClient.WrapKey(ctx, e.KvKey.VaultURL, e.KvKey.KeyName, e.KvKey.KeyVersion, parameters)
	if err != nil {
		return nil, err
	}

	if result.Result == nil {
		return nil, fmt.Errorf("Error when wrapping key with Key Vault for Azure Remote State")
	}

	return result.Result, nil
}

func (e *EncryptionClient) unwrapKey(ctx context.Context, key *string) ([]byte, error) {
	if err := e.validateKeyVaultKey(ctx); err != nil {
		return nil, err
	}

	parameters := keyvault.KeyOperationsParameters{
		Algorithm: keyvault.RSA15,
		Value:     key,
	}

	result, err := e.KvClient.UnwrapKey(ctx, e.KvKey.VaultURL, e.KvKey.KeyName, e.KvKey.KeyVersion, parameters)
	if err != nil {
		return nil, err
	}

	if result.Result == nil {
		return nil, fmt.Errorf("Error when unwrapping key with Key Vault for Azure Remote State")
	}

	decoded, err := base64.RawURLEncoding.DecodeString(*result.Result)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

func (e *EncryptionClient) validateKeyVaultKey(ctx context.Context) error {
	keyBundle, err := e.KvClient.GetKey(ctx, e.KvKey.VaultURL, e.KvKey.KeyName, e.KvKey.KeyVersion)
	if err != nil {
		return err
	}

	return validateKeyVaultBundle(keyBundle)
}

func validateKeyVaultBundle(keyBundle keyvault.KeyBundle) error {
	if keyBundle.Attributes == nil || keyBundle.Key == nil {
		return fmt.Errorf("Azure Key Vault key result contains null values")
	}

	if !*keyBundle.Attributes.Enabled {
		return fmt.Errorf("Azure Key Vault key provided is not enabled. Please enabled it to continue")
	}

	if keyBundle.Key.Kty != keyvault.RSA {
		return fmt.Errorf("Azure Key Vault key provided is not RSA. Please use a RSA key to continue")
	}

	if !stringSliceContains(*keyBundle.Key.KeyOps, "wrapKey") {
		return fmt.Errorf("Azure Key Vault key provided needs to have 'wrapKey' permission. Please set this permission to continue")
	}

	if !stringSliceContains(*keyBundle.Key.KeyOps, "unwrapKey") {
		return fmt.Errorf("Azure Key Vault key provided needs to have 'unwrapKey' permission. Please set this permission to continue")
	}

	return nil
}

func stringSliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func encrypt(plaintext []byte) (ciphertext []byte, key []byte, nonce []byte, err error) {
	key, err = randomSequence(32) // AES-256 requires a 32 bytes key
	if err != nil {
		return nil, nil, nil, err
	}
	nonce, err = randomSequence(12) // GCM mode requires a 12 bytes nonce
	if err != nil {
		return nil, nil, nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, err
	}

	ciphertext = aesgcm.Seal(nil, nonce, plaintext, nil)

	return ciphertext, key, nonce, nil
}

func randomSequence(length int) ([]byte, error) {
	sequence := make([]byte, length)
	_, err := rand.Read(sequence)
	if err != nil {
		return nil, err
	}
	return sequence, nil
}

func decrypt(ciphertext []byte, key []byte, nonce []byte) (plaintext []byte, err error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err = aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
