package docker

import (
	"fmt"
	"os"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

const (
	// DefaultScriptPath is used as the path to copy the file to
	// for remote execution if not provided otherwise.
	DefaultScriptPath = "/tmp/terraform_%RAND%.sh"
)

// connectionInfo is decoded from the ConnInfo of the resource. These are the
// only keys we look at.
type connectionInfo struct {
	Container  string `mapstructure:"container"`
	Host       string `mapstructure:"host"`
	CertPath   string `mapstructure:"cert_path"`
	ScriptPath string `mapstructure:"script_path"`
}

// parseConnectionInfo is used to convert the ConnInfo of the InstanceState into
// a ConnectionInfo struct
func parseConnectionInfo(s *terraform.InstanceState) (*connectionInfo, error) {
	connInfo := &connectionInfo{}
	decConf := &mapstructure.DecoderConfig{
		WeaklyTypedInput: true,
		Result:           connInfo,
	}
	dec, err := mapstructure.NewDecoder(decConf)
	if err != nil {
		return nil, err
	}
	if err := dec.Decode(s.Ephemeral.ConnInfo); err != nil {
		return nil, err
	}

	if connInfo.Host == "" {
		connInfo.Host = os.Getenv("DOCKER_HOST")
	}
	if connInfo.Host == "" {
		connInfo.Host = "unix:///var/run/docker.sock"
	}

	if connInfo.Container == "" {
		return nil, fmt.Errorf("'container' required for docker connection config")
	}

	if connInfo.ScriptPath == "" {
		connInfo.ScriptPath = DefaultScriptPath
	}

	return connInfo, nil
}
