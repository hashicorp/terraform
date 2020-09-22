package azure

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"fmt"
	"math"
	"math/big"
	"regexp"

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

// NewEncryptionClient creates an EncryptionClient from a Key Vault key identifier and a Key Vault client
func NewEncryptionClient(keyVaultKeyIdentifier string, kvClient *keyvault.BaseClient) (*EncryptionClient, error) {
	if kvClient == nil {
		return nil, fmt.Errorf("kvClient cannot be nil")
	}

	kvInfo, err := parseKeyVaultKeyInfo(keyVaultKeyIdentifier)
	if err != nil {
		return nil, err
	}

	return &EncryptionClient{KvClient: kvClient, kvInfo: kvInfo}, nil
}

func parseKeyVaultKeyInfo(keyVaultKeyIdentifier string) (*KeyVaultKeyInfo, error) {
	r, _ := regexp.Compile("https?://(.+)\\.vault\\.azure\\.net/keys/([^\\/.]+)/?([^\\/.]*)")

	str := r.FindStringSubmatch(keyVaultKeyIdentifier)
	if len(str) < 4 {
		return &KeyVaultKeyInfo{}, fmt.Errorf("Expected a key identifier from Key Vault. e.g.: https://keyvaultname.vault.azure.net/keys/myKey/99d67321dd9841af859129cd5551a871")
	}

	info := KeyVaultKeyInfo{}
	info.vaultURL = fmt.Sprintf("https://%s.vault.azure.net", str[1])
	info.keyName = str[2]
	info.keyVersion = str[3]

	return &info, nil
}

func generatePublicKeyFromKeyBundle(keyBundle *keyvault.KeyBundle) (*rsa.PublicKey, error) {
	nDecoded, err := base64.RawURLEncoding.DecodeString(*keyBundle.Key.N)
	if err != nil {
		return nil, err
	}

	eDecoded, err := base64.RawURLEncoding.DecodeString(*keyBundle.Key.E)
	if err != nil {
		return nil, err
	}

	var n, e big.Int
	n.SetBytes(nDecoded)
	e.SetBytes(eDecoded)

	pubKey := rsa.PublicKey{E: int(e.Int64()), N: &n}

	return &pubKey, nil
}

func stringSliceContains(slice []string, str string) bool {
	for _, s := range slice {
		if s == str {
			return true
		}
	}
	return false
}

func validateKeyVaultKeyDetails(keyBundle *keyvault.KeyBundle, pubKey *rsa.PublicKey) error {
	k := pubKey.Size() * 8
	if k != 2048 && k != 3072 && k != 4096 {
		return fmt.Errorf("Azure Key Vault key must have a 2048, 3072 or 4096 length. Please provide a key with other size")
	}

	if !*keyBundle.Attributes.Enabled {
		return fmt.Errorf("Azure Key Vault key provided is not enabled. Please enabled it to continue")
	}

	if keyBundle.Key.Kty != keyvault.RSA {
		return fmt.Errorf("Azure Key Vault key provided is not RSA. Please use a RSA key to continue")
	}

	if !stringSliceContains(*keyBundle.Key.KeyOps, "encrypt") {
		return fmt.Errorf("Azure Key Vault key provided needs to have 'encrypt' permission. Please set this permission to continue")
	}

	if !stringSliceContains(*keyBundle.Key.KeyOps, "decrypt") {
		return fmt.Errorf("Azure Key Vault key provided needs to have 'decrypt' permission. Please set this permission to continue")
	}

	return nil
}

func getKeyVaultAlgorithmParameters(algorithm keyvault.JSONWebKeyEncryptionAlgorithm, keySizeBits int) *KeyVaultAlgorithmParameters {
	kp := &KeyVaultAlgorithmParameters{kvAlgorithm: algorithm, keySizeBits: keySizeBits}

	switch algorithm {
	case keyvault.RSA15:
		kp.encryptBlockSizeBytes = (keySizeBits - 88) / 8
	case keyvault.RSAOAEP:
		kp.encryptBlockSizeBytes = (keySizeBits - 336) / 8
	case keyvault.RSAOAEP256:
		kp.encryptBlockSizeBytes = (keySizeBits - 528) / 8
	default:
		kp.encryptBlockSizeBytes = -1
	}
	kp.decryptBlockSizeBytes = int(math.Ceil(float64(keySizeBits) / 6.0)) // output is base64 encoded ((4 * k / 3) / 8) => (k / 6)

	return kp
}

func (e *EncryptionClient) getKeyVaultKeyDetails(ctx context.Context) (*keyvault.KeyBundle, error) {
	keyBundle, err := e.KvClient.GetKey(ctx, e.kvInfo.vaultURL, e.kvInfo.keyName, e.kvInfo.keyVersion)

	if err != nil {
		return nil, err
	}

	return &keyBundle, nil
}

func (e *EncryptionClient) fillKeyDetails(ctx context.Context) error {
	keyDetails, err := e.getKeyVaultKeyDetails(ctx)
	if err != nil {
		return err
	}

	pubKey, err := generatePublicKeyFromKeyBundle(keyDetails)
	if err != nil {
		return err
	}

	err = validateKeyVaultKeyDetails(keyDetails, pubKey)
	if err != nil {
		return err
	}

	keySizeBits := pubKey.Size() * 8

	e.kvAlgorithmParameters = getKeyVaultAlgorithmParameters(keyvault.RSA15, keySizeBits)

	return nil
}

func (e *EncryptionClient) buildKeyOperationsParameters(value *string) keyvault.KeyOperationsParameters {
	parameters := keyvault.KeyOperationsParameters{}
	parameters.Algorithm = e.kvAlgorithmParameters.kvAlgorithm
	parameters.Value = value
	return parameters
}

func (e *EncryptionClient) encryptByteBlock(ctx context.Context, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	// Lazy loading for key details
	if e.kvAlgorithmParameters == nil {
		if err := e.fillKeyDetails(ctx); err != nil {
			return nil, err
		}
	}

	if len(data) > e.kvAlgorithmParameters.encryptBlockSizeBytes {
		return nil, fmt.Errorf("Can not encrypt more than %v bytes at a time", e.kvAlgorithmParameters.encryptBlockSizeBytes)
	}

	encoded := base64.RawStdEncoding.EncodeToString(data)

	parameters := e.buildKeyOperationsParameters(&encoded)
	result, err := e.KvClient.Encrypt(ctx, e.kvInfo.vaultURL, e.kvInfo.keyName, e.kvInfo.keyVersion, parameters)
	if err != nil {
		return nil, err
	}

	return []byte(*result.Result), nil
}

func (e *EncryptionClient) decryptByteBlock(ctx context.Context, data []byte) ([]byte, error) {
	if len(data) == 0 {
		return data, nil
	}

	// Lazy loading for key details
	if e.kvAlgorithmParameters == nil {
		if err := e.fillKeyDetails(ctx); err != nil {
			return nil, err
		}
	}

	if len(data) > e.kvAlgorithmParameters.decryptBlockSizeBytes {
		return nil, fmt.Errorf("can not decrypt more than %v bytes at a time", e.kvAlgorithmParameters.decryptBlockSizeBytes)
	}

	str := string(data)

	parameters := e.buildKeyOperationsParameters(&str)
	result, err := e.KvClient.Decrypt(ctx, e.kvInfo.vaultURL, e.kvInfo.keyName, e.kvInfo.keyVersion, parameters)
	if err != nil {
		return nil, err
	}

	decoded, err := base64.RawStdEncoding.DecodeString(*result.Result)
	if err != nil {
		return nil, err
	}

	return decoded, nil
}

// Encrypt takes a []byte as input and calls Azure Key Vault to encrypt it using a previous defined asymmetric key
func (e *EncryptionClient) Encrypt(ctx context.Context, data []byte) ([]byte, error) {
	final := make([]byte, 0)

	if len(data) == 0 {
		return final, nil
	}

	// Lazy loading for key details
	if e.kvAlgorithmParameters == nil {
		if err := e.fillKeyDetails(ctx); err != nil {
			return nil, err
		}
	}

	c := e.kvAlgorithmParameters.encryptBlockSizeBytes
	n := len(data) / c
	for i := 0; i < n; i++ {
		d := data[i*c : (i+1)*c]
		res, err := e.encryptByteBlock(ctx, d)
		if err != nil {
			return nil, err
		}
		final = append(final, res...)
	}
	d := data[n*c : len(data)]
	res, err := e.encryptByteBlock(ctx, d)
	if err != nil {
		return nil, err
	}
	final = append(final, res...)
	return final, nil
}

// Decrypt takes a []byte as input and calls Azure Key Vault to decrypt it using a previous defined asymmetric key
func (e *EncryptionClient) Decrypt(ctx context.Context, data []byte) ([]byte, error) {
	final := make([]byte, 0)

	if len(data) == 0 {
		return final, nil
	}

	// Lazy loading for key details
	if e.kvAlgorithmParameters == nil {
		if err := e.fillKeyDetails(ctx); err != nil {
			return nil, err
		}
	}

	c := e.kvAlgorithmParameters.decryptBlockSizeBytes
	n := len(data) / c
	for i := 0; i < n; i++ {
		d := data[i*c : (i+1)*c]
		res, err := e.decryptByteBlock(ctx, d)
		if err != nil {
			return nil, err
		}
		final = append(final, res...)
	}
	d := data[n*c : len(data)]
	res, err := e.decryptByteBlock(ctx, d)
	if err != nil {
		return nil, err
	}
	final = append(final, res...)
	return final, nil
}
