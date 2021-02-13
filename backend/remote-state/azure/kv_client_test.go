package azure

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	"github.com/hashicorp/terraform/helper/acctest"
)

func TestParseKeyVaultKeyInfoValid(t *testing.T) {
	cases := map[string]*KeyVaultKey{
		"https://keyvaultname.vault.azure.net/keys/myKey/99d67321dd9841af859129cd5551a871": {
			VaultURL:   "https://keyvaultname.vault.azure.net",
			KeyName:    "myKey",
			KeyVersion: "99d67321dd9841af859129cd5551a871",
		},
		"https://keyvaultname.vault.azure.net/keys/myKey/99d67321dd9841af859129cd5551a871/": {
			VaultURL:   "https://keyvaultname.vault.azure.net",
			KeyName:    "myKey",
			KeyVersion: "99d67321dd9841af859129cd5551a871",
		},
		"http://abcde.vault.azure.net/keys/myKey/8120938102983": {
			VaultURL:   "https://abcde.vault.azure.net",
			KeyName:    "myKey",
			KeyVersion: "8120938102983",
		},
		"https://keyvaultname.vault.azure.net/keys/myKey/": {
			VaultURL:   "https://keyvaultname.vault.azure.net",
			KeyName:    "myKey",
			KeyVersion: "",
		},
		"https://keyvaultname.vault.azure.net/keys/myKey": {
			VaultURL:   "https://keyvaultname.vault.azure.net",
			KeyName:    "myKey",
			KeyVersion: "",
		},
	}

	for id, c := range cases {
		k, err := parseKeyVaultKey(id)
		if err != nil {
			t.Errorf("Failing during parsing. Error: %v", err)
		}
		if !reflect.DeepEqual(c, k) {
			t.Errorf("Failing during parsing. Expected: %v, Got: %v", c, k)
		}
	}
}

func TestParseKeyVaultKeyInfoInvalid(t *testing.T) {
	errorCases := []string{
		"",
		" ",
		"https://keyvaultname.vault.azure.net",
		"https://keyvaultname.vault.azure.net/",
		"http://keyvaultname.vault.azure.net",
		"http://keyvaultname.vault.azure.net/",
		"https://keyvaultname.vault.azure.net/keys",
		"https://keyvaultname.vault.azure.net/keys/",
		"https://keyvaultname.vault.azure.net/something/myKey/99d67321dd9841af859129cd5551a871",
	}

	for _, id := range errorCases {
		k, err := parseKeyVaultKey(id)
		if err == nil {
			t.Errorf("Failing during parsing. It should not parse an error case. Got: %v", k)
		}
	}
}

func TestCreateEncryptClientValid(t *testing.T) {
	keyVaultKeyIdentifier := "https://keyvaultname.vault.azure.net/keys/myKey/99d67321dd9841af859129cd5551a871"
	kvClient := &keyvault.BaseClient{}

	_, err := NewEncryptionClient(keyVaultKeyIdentifier, kvClient)

	if err != nil {
		t.Errorf("Error when creating EncryptionClient: %v", err)
	}
}

func TestCreateEncryptClientInvalid(t *testing.T) {
	keyVaultKeyIdentifier := "https://keyvaultname.vault.azure.net/keys/myKey/99d67321dd9841af859129cd5551a871"

	_, err := NewEncryptionClient(keyVaultKeyIdentifier, nil)

	if err == nil {
		t.Errorf("Error when creating EncryptionClient. Expected an error.")
	}
}

func TestKeyVaultKeyBundleValidation(t *testing.T) {
	isEnabled := true
	keyID := "https://vaultname.vault.azure.net/keys/key2048/b5c419ae7aa847459c5f72719dfe522e"
	keyOps := []string{"wrapKey", "unwrapKey"}
	keyType := keyvault.RSA
	n := "t8zB6yX6fIyloMqhfW07ihq0x22OdyW5h2gX2QjCN6O5cko21at_7f2tCjrBdO1bJs_Zr5nIjWKU3vJTGCJNijfrOMNIoamMxJumRpDraDiYD6e41CVw-oTHW4l8Yb_k81Dph9Ojfar9FlIJ-hTsIKkNb2laeLDgYqMKmtJIoXEsMUKKqzzC5-fDt_Q5Tj-ETJoNDiSgvmbfmwJDf2r2wIES_UJ8jY2sRcaa9YaHmgsA-NCHDH2Rs4ssU4f7TaEuyTsUQlNifKrME0AtKKv8do23g8_BTbIu1uFCGDxG-kBt83mCn27raT3lbo9P6ZKJmyGRAHIhJehA_2v3vpC30Q"
	e := "AQAB"

	t.Run("KeyValid", func(t *testing.T) {
		keyBundle := &keyvault.KeyBundle{
			Attributes: &keyvault.KeyAttributes{Enabled: &isEnabled},
			Key: &keyvault.JSONWebKey{
				Kid:    &keyID,
				KeyOps: &keyOps,
				Kty:    keyType,
				N:      &n,
				E:      &e,
			},
		}

		err := validateKeyVaultBundle(*keyBundle)

		if err != nil {
			t.Errorf("Error when validating key details. Got an error: %v", err)
		}
	})

	t.Run("NotEnabled", func(t *testing.T) {
		isEnabled := false

		keyBundle := &keyvault.KeyBundle{
			Attributes: &keyvault.KeyAttributes{Enabled: &isEnabled},
			Key: &keyvault.JSONWebKey{
				Kid:    &keyID,
				KeyOps: &keyOps,
				Kty:    keyType,
				N:      &n,
				E:      &e,
			},
		}

		err := validateKeyVaultBundle(*keyBundle)

		msg := "Azure Key Vault key provided is not enabled"
		if !strings.Contains(err.Error(), msg) {
			t.Errorf("Expected a different error. Expected: %v, Got: %v", msg, err.Error())
		}
	})

	t.Run("NotRSA", func(t *testing.T) {
		keyType := keyvault.EC

		keyBundle := &keyvault.KeyBundle{
			Attributes: &keyvault.KeyAttributes{Enabled: &isEnabled},
			Key: &keyvault.JSONWebKey{
				Kid:    &keyID,
				KeyOps: &keyOps,
				Kty:    keyType,
				N:      &n,
				E:      &e,
			},
		}

		err := validateKeyVaultBundle(*keyBundle)

		msg := "Azure Key Vault key provided is not RSA"
		if !strings.Contains(err.Error(), msg) {
			t.Errorf("Expected a different error. Expected: %v, Got: %v", msg, err.Error())
		}
	})

	t.Run("MissingWrapKeyPermissions", func(t *testing.T) {
		keyOps := []string{"unwrapKey"}

		keyBundle := &keyvault.KeyBundle{
			Attributes: &keyvault.KeyAttributes{Enabled: &isEnabled},
			Key: &keyvault.JSONWebKey{
				Kid:    &keyID,
				KeyOps: &keyOps,
				Kty:    keyType,
				N:      &n,
				E:      &e,
			},
		}

		err := validateKeyVaultBundle(*keyBundle)

		msg := "Azure Key Vault key provided needs to have 'wrapKey' permission"
		if !strings.Contains(err.Error(), msg) {
			t.Errorf("Expected a different error. Expected: %v, Got: %v", msg, err.Error())
		}
	})

	t.Run("MissingUnwrapKeyPermissions", func(t *testing.T) {
		keyOps := []string{"wrapKey"}

		keyBundle := &keyvault.KeyBundle{
			Attributes: &keyvault.KeyAttributes{Enabled: &isEnabled},
			Key: &keyvault.JSONWebKey{
				Kid:    &keyID,
				KeyOps: &keyOps,
				Kty:    keyType,
				N:      &n,
				E:      &e,
			},
		}

		err := validateKeyVaultBundle(*keyBundle)

		msg := "Azure Key Vault key provided needs to have 'unwrapKey' permission"
		if !strings.Contains(err.Error(), msg) {
			t.Errorf("Expected a different error. Expected: %v, Got: %v", msg, err.Error())
		}
	})
}

func TestEncryption(t *testing.T) {
	dataStr := acctest.RandString(2 * 1024 * 1024) // 2 M chars * 4 B/char = 8 MB
	data := []byte(dataStr)

	encrypted, key, nonce, err := encrypt(data)
	if err != nil {
		t.Errorf("Error when encrypting data. Got the error: %v", err)
	}

	decrypted, err := decrypt(encrypted, key, nonce)
	if err != nil {
		t.Errorf("Error when decrypting data. Got the error: %v", err)
	}

	if !reflect.DeepEqual(decrypted, data) {
		t.Fatalf("encrypting/decrypting was not correct. Expected: %v, Got: %v", data, decrypted)
	}
}

func TestAccKeyVaultOperations(t *testing.T) {
	testAccAzureBackend(t)

	ctx := context.TODO()
	rs := acctest.RandString(4)

	keyVaultName := fmt.Sprintf("keyvaultterraform%s", rs)
	keyName := "myKey"

	res := testResourceNamesWithKeyVault(rs, "testState", keyVaultName, keyName)
	armClient := buildTestClient(t, res)

	defer armClient.destroyTestResources(ctx, res)
	err := armClient.buildTestResources(ctx, &res)
	if err != nil {
		t.Errorf("Error creating Test Resources: %q", err)
	}

	c := armClient.encClient

	dataStr := acctest.RandString(2 * 1024 * 1024) // 2 M chars * 4 B/char = 8 MB
	data := []byte(dataStr)

	t.Run("GetKey", func(t *testing.T) {
		err := c.validateKeyVaultKey(ctx)
		if err != nil {
			t.Fatalf("Error when getting key: %v", err)
		}
	})

	t.Run("WrapUnwrapKey", func(t *testing.T) {
		key := []byte("ILWZQuq2jiUY9ycMYiT8uymt9zSEtTyf")

		wrapped, err := c.wrapKey(ctx, key)
		if err != nil {
			t.Fatalf("Error when wrapping key: %v", err)
		}

		unwrapped, err := c.unwrapKey(ctx, wrapped)
		if err != nil {
			t.Fatalf("Error when unwrapping key: %v", err)
		}

		if !reflect.DeepEqual(key, unwrapped) {
			t.Fatalf("Data received from decryption was not correct. Expected: %v, Got: %v", wrapped, unwrapped)
		}
	})

	t.Run("EncryptDecrypt", func(t *testing.T) {
		encrypted, err := c.Encrypt(ctx, data)
		if err != nil {
			t.Fatalf("Error when encrypting data: %v", err)
		}

		decrypted, err := c.Decrypt(ctx, encrypted)
		if err != nil {
			t.Fatalf("Error when decrypting data: %v", err)
		}

		if !reflect.DeepEqual(decrypted, data) {
			t.Fatalf("Data received from decryption was not correct. Expected: %v, Got: %v", data, decrypted)
		}
	})
}
