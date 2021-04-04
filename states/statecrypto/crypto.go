package statecrypto

import (
	"github.com/hashicorp/terraform/states/statecrypto/implementations/aes256state"
	"github.com/hashicorp/terraform/states/statecrypto/statecryptoif"
	"log"
	"os"
	"strings"
)

// Set this environment variable to "ALGORITHM:KEY". Supported algorithms:
//   AES256, then KEY must be exactly 64 hexadecimal lower case characters for the 32 byte AES256 key
var KeyEnvName = "TF_REMOTE_STATE_ENCRYPTION"

func configuration() []string {
	return strings.Split(os.Getenv(KeyEnvName), ":")
}

func configuredImplementation(config []string) string {
	if len(config) > 0 {
		return config[0]
	}
	return ""
}

func StateCrypto() statecryptoif.StateCrypto {
	var implementation statecryptoif.StateCrypto
	var err error = nil

	config := configuration()
	switch name := configuredImplementation(config); name {
	case "AES256":
		implementation, err = aes256state.New(config)
	default:
		implementation = nil
	}

	if err != nil {
		// TODO how to correctly handle configuration errors?
		log.Fatalf("error configuring state file crypto: %v", err)
	}

	return implementation
}
