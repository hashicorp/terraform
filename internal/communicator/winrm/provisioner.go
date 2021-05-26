package winrm

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/communicator/shared"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

const (
	// DefaultUser is used if there is no user given
	DefaultUser = "Administrator"

	// DefaultPort is used if there is no port given
	DefaultPort = 5985

	// DefaultHTTPSPort is used if there is no port given and HTTPS is true
	DefaultHTTPSPort = 5986

	// DefaultScriptPath is used as the path to copy the file to
	// for remote execution if not provided otherwise.
	DefaultScriptPath = "C:/Temp/terraform_%RAND%.cmd"

	// DefaultTimeout is used if there is no timeout given
	DefaultTimeout = 5 * time.Minute
)

// connectionInfo is decoded from the ConnInfo of the resource. These are the
// only keys we look at. If a KeyFile is given, that is used instead
// of a password.
type connectionInfo struct {
	User       string
	Password   string
	Host       string
	Port       uint16
	HTTPS      bool
	Insecure   bool
	NTLM       bool   `mapstructure:"use_ntlm"`
	CACert     string `mapstructure:"cacert"`
	Timeout    string
	ScriptPath string        `mapstructure:"script_path"`
	TimeoutVal time.Duration `mapstructure:"-"`
}

// decodeConnInfo decodes the given cty.Value using the same behavior as the
// lgeacy mapstructure decoder in order to preserve as much of the existing
// logic as possible for compatibility.
func decodeConnInfo(v cty.Value) (*connectionInfo, error) {
	connInfo := &connectionInfo{}
	if v.IsNull() {
		return connInfo, nil
	}

	for k, v := range v.AsValueMap() {
		if v.IsNull() {
			continue
		}

		switch k {
		case "user":
			connInfo.User = v.AsString()
		case "password":
			connInfo.Password = v.AsString()
		case "host":
			connInfo.Host = v.AsString()
		case "port":
			if err := gocty.FromCtyValue(v, &connInfo.Port); err != nil {
				return nil, err
			}
		case "https":
			connInfo.HTTPS = v.True()
		case "insecure":
			connInfo.Insecure = v.True()
		case "use_ntlm":
			connInfo.NTLM = v.True()
		case "cacert":
			connInfo.CACert = v.AsString()
		case "script_path":
			connInfo.ScriptPath = v.AsString()
		case "timeout":
			connInfo.Timeout = v.AsString()
		}
	}
	return connInfo, nil
}

// parseConnectionInfo is used to convert the ConnInfo of the InstanceState into
// a ConnectionInfo struct
func parseConnectionInfo(v cty.Value) (*connectionInfo, error) {
	v, err := shared.ConnectionBlockSupersetSchema.CoerceValue(v)
	if err != nil {
		return nil, err
	}

	connInfo, err := decodeConnInfo(v)
	if err != nil {
		return nil, err
	}
	// Check on script paths which point to the default Windows TEMP folder because files
	// which are put in there very early in the boot process could get cleaned/deleted
	// before you had the change to execute them.
	//
	// TODO (SvH) Needs some more debugging to fully understand the exact sequence of events
	// causing this...
	if strings.HasPrefix(filepath.ToSlash(connInfo.ScriptPath), "C:/Windows/Temp") {
		return nil, fmt.Errorf(
			`Using the C:\Windows\Temp folder is not supported. Please use a different 'script_path'.`)
	}

	if connInfo.User == "" {
		connInfo.User = DefaultUser
	}

	// Format the host if needed.
	// Needed for IPv6 support.
	connInfo.Host = shared.IpFormat(connInfo.Host)

	if connInfo.Port == 0 {
		if connInfo.HTTPS {
			connInfo.Port = DefaultHTTPSPort
		} else {
			connInfo.Port = DefaultPort
		}
	}
	if connInfo.ScriptPath == "" {
		connInfo.ScriptPath = DefaultScriptPath
	}
	if connInfo.Timeout != "" {
		connInfo.TimeoutVal = safeDuration(connInfo.Timeout, DefaultTimeout)
	} else {
		connInfo.TimeoutVal = DefaultTimeout
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

func formatDuration(duration time.Duration) string {
	h := int(duration.Hours())
	m := int(duration.Minutes()) - h*60
	s := int(duration.Seconds()) - (h*3600 + m*60)

	res := "PT"
	if h > 0 {
		res = fmt.Sprintf("%s%dH", res, h)
	}
	if m > 0 {
		res = fmt.Sprintf("%s%dM", res, m)
	}
	if s > 0 {
		res = fmt.Sprintf("%s%dS", res, s)
	}

	return res
}
