package docker

import (
	"log"
	"time"

	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

const (
	// DefaultTimeout is used if there is no timeout given
	DefaultTimeout = 5 * time.Minute

	// DefaultScriptPath is used as the path to copy the file to
	// for remote execution if not provided otherwise.
	DefaultScriptPath = "/tmp/terraform_%RAND%.sh"
)

// connectionInfo is decoded from the ConnInfo of the resource. These are the
// only keys we look at. If a KeyFile is given, that is used instead
// of a password.
type connectionInfo struct {
	User        string
	Password    string
	CertPath    string `mapstructure:"cert_path"`
	Host        string
	Port        int
	Agent       bool
	Timeout     string
	ScriptPath  string        `mapstructure:"script_path"`
	TimeoutVal  time.Duration `mapstructure:"-"`
	ContainerId string
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
	if connInfo.Timeout != "" {
		connInfo.TimeoutVal = safeDuration(connInfo.Timeout, DefaultTimeout)
	} else {
		connInfo.TimeoutVal = DefaultTimeout
	}
	if connInfo.ContainerId == "" {
		connInfo.ContainerId = s.ID
	}
	if connInfo.ScriptPath == "" {
		connInfo.ScriptPath = DefaultScriptPath
	}

	return connInfo, nil
}

// safeDuration returns either the parsed duration or a default value
func safeDuration(dur string, defaultDur time.Duration) time.Duration {
	d, err := time.ParseDuration(dur)
	if err != nil {
		log.Printf("Invalid duration '%s', using default of %s", dur, defaultDur)
		return defaultDur
	}
	return d
}
