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

// Unit Tests

func TestParseKeyVaultKeyInfoValid(t *testing.T) {
	testAccAzureBackend(t)

	cases := map[string]*KeyVaultKeyInfo{
		"https://keyvaultname.vault.azure.net/keys/myKey/99d67321dd9841af859129cd5551a871": &KeyVaultKeyInfo{
			vaultURL:   "https://keyvaultname.vault.azure.net",
			keyName:    "myKey",
			keyVersion: "99d67321dd9841af859129cd5551a871",
		},
		"https://keyvaultname.vault.azure.net/keys/myKey/99d67321dd9841af859129cd5551a871/": &KeyVaultKeyInfo{
			vaultURL:   "https://keyvaultname.vault.azure.net",
			keyName:    "myKey",
			keyVersion: "99d67321dd9841af859129cd5551a871",
		},
		"http://abcde.vault.azure.net/keys/myKey/8120938102983": &KeyVaultKeyInfo{
			vaultURL:   "https://abcde.vault.azure.net",
			keyName:    "myKey",
			keyVersion: "8120938102983",
		},
		"https://keyvaultname.vault.azure.net/keys/myKey/": &KeyVaultKeyInfo{
			vaultURL:   "https://keyvaultname.vault.azure.net",
			keyName:    "myKey",
			keyVersion: "",
		},
		"https://keyvaultname.vault.azure.net/keys/myKey": &KeyVaultKeyInfo{
			vaultURL:   "https://keyvaultname.vault.azure.net",
			keyName:    "myKey",
			keyVersion: "",
		},
	}

	for id, c := range cases {
		k, err := parseKeyVaultKeyInfo(id)
		if err != nil {
			t.Errorf("Failing during parsing. Error: %v", err)
		}
		if !reflect.DeepEqual(c, k) {
			t.Errorf("Failing during parsing. Expected: %v, Got: %v", c, k)
		}
	}
}

func TestParseKeyVaultKeyInfoInvalid(t *testing.T) {
	testAccAzureBackend(t)

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
		k, err := parseKeyVaultKeyInfo(id)
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

func TestGeneratePublicKeyFromKeyBundle(t *testing.T) {
	testAccAzureBackend(t)

	cases := []struct {
		n        string
		e        string
		sizeBits int
	}{
		{
			n:        "t8zB6yX6fIyloMqhfW07ihq0x22OdyW5h2gX2QjCN6O5cko21at_7f2tCjrBdO1bJs_Zr5nIjWKU3vJTGCJNijfrOMNIoamMxJumRpDraDiYD6e41CVw-oTHW4l8Yb_k81Dph9Ojfar9FlIJ-hTsIKkNb2laeLDgYqMKmtJIoXEsMUKKqzzC5-fDt_Q5Tj-ETJoNDiSgvmbfmwJDf2r2wIES_UJ8jY2sRcaa9YaHmgsA-NCHDH2Rs4ssU4f7TaEuyTsUQlNifKrME0AtKKv8do23g8_BTbIu1uFCGDxG-kBt83mCn27raT3lbo9P6ZKJmyGRAHIhJehA_2v3vpC30Q",
			e:        "AQAB",
			sizeBits: 2048,
		},
		{
			n:        "m9rXO6iYmBMx-57d-r1DlsNBnFLRPyifD2Ax7d6-W4AqyvTZ9Ac5aII4ds06eKH7bHg7fxIrnznKd3tiA4WhRctonPx2TJLQ95m_VbCArdkmIctfH-Rn_-Jg5ynj9kUgJUFNhRHw3DJEwI9aV6zogiuTtO47J5Nwg6I9G4BzRpOu7Hkp5SrA4XUwr9ZTy3wJJFee3O7AfCtoi196JsnvZxsNismIoQoO4R0eYzQQC-H1tjSwIlvfFJNQA2PRG5Ev3Tkk4rrAsO0rvSvjMC1am321P2acpW5gan1lXNJbChzY2fCLTysFtQVGQWgWSePNE1NA3jppsr5JA2moCq_g7hfEqAFuz4leqqbGZwVxz7rDwYFL-yQC7rffV4dy4qAIlTWLNRBXNUXgJXvJdABAbdWkxP7HSTvdWz92mpSpztcdC2Sx_wY_kZO5SCMdw6p3iCJOVKOuYqfOmACinc7Q-X190XVel3OGZdTTBGfPYS6vdRfKi1TZHr2FO69FNbdL",
			e:        "AQAB",
			sizeBits: 3072,
		},
		{
			n:        "9aICHHp3OzKUU9ttEfKj14956PVhC_XWThJhFHVqILUxretGORdch13AeuhVf0qYMHedeVkDBehU24rn_9oc11rXxYFUEjAsrk-uhJBVmGr3ZCFwylle0ZQ-fHIigPhYauPz2g87fQsLenfyxl4r2i5xfQun2c4hHFlsaUanhX4P4m8pQTi3hRM-q1ZeP3dF9NmIAmrd6pSQ9MpCalFdp4Ic71kpHl45McHEV40DeivWJD4wnugF35LQcvOmTjnEiHkp_OyaYJ7hKhGiLd2NL0NlIerLe9-QjIgY4aqmgIzN4Er9d6sKaHS3QVW9ocj08WwRHVOhQTZrCGU6rUei_ygRZKESFZ_Jd8IDBoD9Vnza1xSerGntRsooOZvuexcFlRpvspGB0zPchygWiF7QTQvVwjTBqiUCP7DUyUTg9sgpNf6B3T-LxMzuIH4rQxlhnxiYjPwQtDAQrIFNjZLWqN_uor7mvaH9kuy01NElCa2Pijy4sLa9qxfq37g0L7y7jJLtcxWhOlhN2jfCWsmCMIGU5n4g08Tc-rf1bQBHCNyYohgPUDWdj0PYnas4nZ3fX1sijOwKpyZIpTgd-OoyTfjcrNVsnmVKT6xt6dBqS1bz53CeOgUahsaubcvflt68hAT09KyQTyG8aH5cb0nNICf5EZnVctAklP4x09Qv2tk",
			e:        "AQAB",
			sizeBits: 4096,
		},
	}

	for _, k := range cases {
		keyBundle := &keyvault.KeyBundle{
			Key: &keyvault.JSONWebKey{
				N: &k.n,
				E: &k.e,
			},
		}

		pubKey, err := generatePublicKeyFromKeyBundle(keyBundle)
		if err != nil {
			t.Errorf("Error when generating public key. Got an error: %v", err)
		}

		if pubKey.E != 65537 {
			t.Errorf("Error when validating public key. Expected: %v Got: %v", 65537, pubKey.E)
		}

		size := pubKey.Size() * 8
		if size != k.sizeBits {
			t.Errorf("Error when validating public key size. Expected: %v Got: %v", k.sizeBits, size)
		}
	}
}

func TestValidateKeyVaultKeyDetails(t *testing.T) {
	testAccAzureBackend(t)

	isEnabled := true
	keyID := "https://vaultname.vault.azure.net/keys/key2048/b5c419ae7aa847459c5f72719dfe522e"
	keyOps := []string{"encrypt", "decrypt"}
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

		pubKey, err := generatePublicKeyFromKeyBundle(keyBundle)
		if err != nil {
			t.Errorf("Error when generating public key. Got an error: %v", err)
		}

		err = validateKeyVaultKeyDetails(keyBundle, pubKey)
		if err != nil {
			t.Errorf("Error when validating key details. Got an error: %v", err)
		}
	})

	t.Run("SizeNotAllowed", func(t *testing.T) {
		n := "8iFM9gXomFikqW3K4Svw58MRPU-FYZaNt-QUwmrl9qywsukjlY167DE5Zo4v25nsb5r3YhjOqzjjqKkFXUJNFRrEuSwGGp1n6NRbo-S8Jsf9ucwj7p0wSY_U9gFMJlYH0gD-jTlQkQ0fBHFdAIK9LaZs5ZeFplxjCTqTQg3AxpU"

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

		pubKey, err := generatePublicKeyFromKeyBundle(keyBundle)
		if err != nil {
			t.Errorf("Error when generating public key. Got an error: %v", err)
		}

		err = validateKeyVaultKeyDetails(keyBundle, pubKey)
		msg := "Azure Key Vault key must have a 2048, 3072 or 4096 length"
		if !strings.Contains(err.Error(), msg) {
			t.Errorf("Expected a different error. Expected: %v, Got: %v", msg, err.Error())
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

		pubKey, err := generatePublicKeyFromKeyBundle(keyBundle)
		if err != nil {
			t.Errorf("Error when generating public key. Got an error: %v", err)
		}

		err = validateKeyVaultKeyDetails(keyBundle, pubKey)
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

		pubKey, err := generatePublicKeyFromKeyBundle(keyBundle)
		if err != nil {
			t.Errorf("Error when generating public key. Got an error: %v", err)
		}

		err = validateKeyVaultKeyDetails(keyBundle, pubKey)
		msg := "Azure Key Vault key provided is not RSA"
		if !strings.Contains(err.Error(), msg) {
			t.Errorf("Expected a different error. Expected: %v, Got: %v", msg, err.Error())
		}
	})

	t.Run("MissingEncryptPermissions", func(t *testing.T) {
		keyOps := []string{"decrypt"}

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

		pubKey, err := generatePublicKeyFromKeyBundle(keyBundle)
		if err != nil {
			t.Errorf("Error when generating public key. Got an error: %v", err)
		}

		err = validateKeyVaultKeyDetails(keyBundle, pubKey)
		msg := "Azure Key Vault key provided needs to have 'encrypt' permission"
		if !strings.Contains(err.Error(), msg) {
			t.Errorf("Expected a different error. Expected: %v, Got: %v", msg, err.Error())
		}
	})

	t.Run("MissingDecryptPermissions", func(t *testing.T) {
		keyOps := []string{"encrypt"}

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

		pubKey, err := generatePublicKeyFromKeyBundle(keyBundle)
		if err != nil {
			t.Errorf("Error when generating public key. Got an error: %v", err)
		}

		err = validateKeyVaultKeyDetails(keyBundle, pubKey)
		msg := "Azure Key Vault key provided needs to have 'decrypt' permission"
		if !strings.Contains(err.Error(), msg) {
			t.Errorf("Expected a different error. Expected: %v, Got: %v", msg, err.Error())
		}
	})
}

func TestGetKeyVaultAlgorithmParameters(t *testing.T) {
	testAccAzureBackend(t)

	t.Run("RSA15", func(t *testing.T) {
		expected := map[int]*KeyVaultAlgorithmParameters{
			2048: &KeyVaultAlgorithmParameters{encryptBlockSizeBytes: 245, decryptBlockSizeBytes: 342},
			3072: &KeyVaultAlgorithmParameters{encryptBlockSizeBytes: 373, decryptBlockSizeBytes: 512},
			4096: &KeyVaultAlgorithmParameters{encryptBlockSizeBytes: 501, decryptBlockSizeBytes: 683},
		}

		testGetKeyVaultAlgorithmParametersWithAlgorithm(t, keyvault.RSA15, expected)
	})

	t.Run("RSAOAEP", func(t *testing.T) {
		expected := map[int]*KeyVaultAlgorithmParameters{
			2048: &KeyVaultAlgorithmParameters{encryptBlockSizeBytes: 214, decryptBlockSizeBytes: 342},
			3072: &KeyVaultAlgorithmParameters{encryptBlockSizeBytes: 342, decryptBlockSizeBytes: 512},
			4096: &KeyVaultAlgorithmParameters{encryptBlockSizeBytes: 470, decryptBlockSizeBytes: 683},
		}

		testGetKeyVaultAlgorithmParametersWithAlgorithm(t, keyvault.RSAOAEP, expected)
	})

	t.Run("RSAOAEP256", func(t *testing.T) {
		expected := map[int]*KeyVaultAlgorithmParameters{
			2048: &KeyVaultAlgorithmParameters{encryptBlockSizeBytes: 190, decryptBlockSizeBytes: 342},
			3072: &KeyVaultAlgorithmParameters{encryptBlockSizeBytes: 318, decryptBlockSizeBytes: 512},
			4096: &KeyVaultAlgorithmParameters{encryptBlockSizeBytes: 446, decryptBlockSizeBytes: 683},
		}

		testGetKeyVaultAlgorithmParametersWithAlgorithm(t, keyvault.RSAOAEP256, expected)
	})
}

func testGetKeyVaultAlgorithmParametersWithAlgorithm(t *testing.T, algorithm keyvault.JSONWebKeyEncryptionAlgorithm, expected map[int]*KeyVaultAlgorithmParameters) {
	testAccAzureBackend(t)

	for size, pe := range expected {
		pa := getKeyVaultAlgorithmParameters(algorithm, size)

		if pa.keySizeBits != size {
			t.Errorf("Key size wasn't correct. Expected: %v, Got: %v", size, pa.keySizeBits)
		}
		if pa.kvAlgorithm != algorithm {
			t.Errorf("Key algorithm wasn't correct. Expected: %v, Got: %v", algorithm, pa.kvAlgorithm)
		}
		if pa.encryptBlockSizeBytes != pe.encryptBlockSizeBytes {
			t.Errorf("Encrypt block size wasn't correct. Expected: %v, Got: %v", pe.encryptBlockSizeBytes, pa.encryptBlockSizeBytes)
		}
		if pa.decryptBlockSizeBytes != pe.decryptBlockSizeBytes {
			t.Errorf("Decrypt block size wasn't correct. Expected: %v, Got: %v", pe.decryptBlockSizeBytes, pa.decryptBlockSizeBytes)
		}
	}
}

// Integration Tests

func TestKeyVaultEncryption(t *testing.T) {
	testAccAzureBackend(t)

	ctx := context.TODO()
	rs := acctest.RandString(4)

	keyVaultName := fmt.Sprintf("keyvaultterraform%s", rs)
	keyName := "myKey"
	keyIdentifier := fmt.Sprintf("https://%s.vault.azure.net/keys/%s", keyVaultName, keyName)

	res := testResourceNamesWithKeyVault(rs, "testState", keyVaultName, keyName)
	armClient := buildTestClient(t, res)

	defer armClient.destroyTestResources(ctx, res)
	err := armClient.buildTestResources(ctx, &res)
	if err != nil {
		t.Errorf("Error creating Test Resources: %q", err)
	}

	c := armClient.encClient

	smallData := acctest.RandString(20)
	largeData := acctest.RandString(3000)

	smallDataBytes := []byte(smallData)
	largeDataBytes := []byte(largeData)

	t.Run("GetDetails", func(t *testing.T) {
		details, err := c.getKeyVaultKeyDetails(ctx)
		if err != nil {
			t.Fatalf("Error when getting key details: %v", err)
		}

		if !strings.Contains(*details.Key.Kid, keyIdentifier) {
			t.Fatalf("Key details was not correct. Expected: %v, Got: %v", keyIdentifier, *details.Key.Kid)
		}
	})

	t.Run("EncryptByteBlock", func(t *testing.T) {
		encrypted, err := c.encryptByteBlock(ctx, smallDataBytes)
		if err != nil {
			t.Fatalf("Error when encrypting data: %v", err)
		}

		if len(encrypted) != 342 { //TODO: Use other key sizes rather than 2048
			t.Fatalf("Error when encrypting data, wrong size. Expected: %v, Got: %v", 342, len(encrypted))
		}
	})

	t.Run("DecryptByteBlock", func(t *testing.T) {
		encrypted, err := c.encryptByteBlock(ctx, smallDataBytes)
		if err != nil {
			t.Fatalf("Error when encrypting data: %v", err)
		}

		decrypted, err := c.decryptByteBlock(ctx, encrypted)
		if err != nil {
			t.Fatalf("Error when decrypting data: %v", err)
		}

		if !reflect.DeepEqual(decrypted, smallDataBytes) {
			t.Fatalf("Data received from decryption was not correct. Expected: %v, Got: %v", smallDataBytes, decrypted)
		}
	})

	t.Run("Encrypt", func(t *testing.T) {
		_, err := c.Encrypt(ctx, largeDataBytes)
		if err != nil {
			t.Fatalf("Error when encrypting data: %v", err)
		}
	})

	t.Run("Decrypt", func(t *testing.T) {
		encrypted, err := c.Encrypt(ctx, largeDataBytes)
		if err != nil {
			t.Fatalf("Error when encrypting data: %v", err)
		}

		decrypted, err := c.Decrypt(ctx, encrypted)
		if err != nil {
			t.Fatalf("Error when decrypting data: %v", err)
		}

		if !reflect.DeepEqual(decrypted, largeDataBytes) {
			t.Fatalf("Data received from decryption was not correct. Expected: %v, Got: %v", largeDataBytes, decrypted)
		}
	})
}
