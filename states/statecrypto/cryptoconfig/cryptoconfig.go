package cryptoconfig

import (
	"encoding/json"
	"log"
	"os"
)

// StateCryptoConfig holds the configuration for transparent client-side remote state encryption
type StateCryptoConfig struct {
	// select the implementation to use
	//
	// supported values are
	//   "client-side/AES256-cfb/SHA256"
	//   "client-side/AES256-gcm"
	//   "azure-key-vault/RSA1.5/AES256-gcm"
	//
	// supplying an unsupported value raises an error
	Implementation string `json:"implementation"`

	// implementation-specific parameters, such as the key
	Parameters map[string]string `json:"parameters"`
}

// TODO mark as experimental feature for now

// The name of the environment variable used to configure encryption and decryption
//
// Set this environment variable to a json representation of StateCryptoConfig, or leave it unset/blank
// to disable encryption.
var ConfigEnvName = "TF_REMOTE_STATE_ENCRYPTION"

// The name of the environment variable used to configure fallback decryption
//
// Set this environment variable to a json representation of StateCryptoConfig, or leave it unset/blank.
//
// Note that decryption will always try the decryption key specified in TF_REMOTE_STATE_ENCRYPTION first.
// If decryption fails with that, it will try this configuration.
//
// Why is this useful?
// - key rotation (put the old key here until all state has been migrated)
// - decryption (leave TF_REMOTE_STATE_ENCRYPTION blank/unset, but set this variable, and your state will be decrypted on next write)
var FallbackConfigEnvName = "TF_REMOTE_STATE_DECRYPTION_FALLBACK"

func parse(setting string) StateCryptoConfig {
	if setting == "" {
		return StateCryptoConfig{}
	}

	config := StateCryptoConfig{}
	err := json.Unmarshal([]byte(setting), &config)
	if err != nil {
		// TODO better handling
		log.Fatalf("error parsing state crypto configuration: %v", err.Error())
	}
	return config
}

func Configuration() StateCryptoConfig {
	return parse(os.Getenv(ConfigEnvName))
}

func FallbackConfiguration() StateCryptoConfig {
	return parse(os.Getenv(FallbackConfigEnvName))
}
