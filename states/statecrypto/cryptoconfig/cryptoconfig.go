package cryptoconfig

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
)

const ClientSide_Aes256cfb_Sha256 = "client-side/AES256-cfb/SHA256"

// StateCryptoConfig holds the configuration for transparent client-side remote state encryption
type StateCryptoConfig struct {
	// Implementation selects the implementation to use
	//
	// supported values are
	//   "client-side/AES256-cfb/SHA256"
	//   "" (means not encrypted, the default)
	//
	// supplying an unsupported value raises an error
	Implementation string `json:"implementation"`

	// Parameters contains implementation-specific parameters, such as the key
	Parameters map[string]string `json:"parameters"`
}

// ConfigEnvName configures the name of the environment variable used to configure encryption and decryption
//
// Set this environment variable to a json representation of StateCryptoConfig, or leave it unset/blank
// to disable encryption.
var ConfigEnvName = "TF_REMOTE_STATE_ENCRYPTION"

// FallbackConfigEnvName configures the name of the environment variable used to configure fallback decryption
//
// Set this environment variable to a json representation of StateCryptoConfig, or leave it unset/blank
// in order to not supply a fallback.
//
// Note that decryption will always try the configuration specified in TF_REMOTE_STATE_ENCRYPTION first.
// Only if decryption fails with that, it will try this configuration.
//
// Why is this useful?
// - key rotation (put the old key here until all state has been migrated)
// - decryption (leave TF_REMOTE_STATE_ENCRYPTION blank/unset, but set this variable, and your state will be decrypted on next write)
var FallbackConfigEnvName = "TF_REMOTE_STATE_DECRYPTION_FALLBACK"

func Configuration() StateCryptoConfig {
	return configFromEnv(ConfigEnvName)
}

func FallbackConfiguration() StateCryptoConfig {
	return configFromEnv(FallbackConfigEnvName)
}

var logFatalf = log.Fatalf

func configFromEnv(envName string) StateCryptoConfig {
	config, err := Parse(os.Getenv(envName))
	if err != nil {
		logFatalf("error parsing remote state encryption configuration from environment variable %s: %s", envName, err.Error())
	}
	return config
}

func Parse(jsonConfig string) (StateCryptoConfig, error) {
	if jsonConfig == "" {
		return StateCryptoConfig{}, nil
	}

	config := StateCryptoConfig{}

	dec := json.NewDecoder(bytes.NewReader([]byte(jsonConfig)))
	dec.DisallowUnknownFields()
	err := dec.Decode(&config)

	return config, err
}
