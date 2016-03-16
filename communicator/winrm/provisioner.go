package winrm

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/pathorcontents"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/mapstructure"
)

const (
	// DefaultUser is used if there is no user given
	DefaultUser = "Administrator"

	// DefaultPort is used if there is no port given
	DefaultPort = 5985

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
	User               string
	Password           string
	PasswordPrivateKey string `mapstructure:"password_private_key"`
	Host               string
	Port               int
	HTTPS              bool
	Insecure           bool
	CACert             *[]byte `mapstructure:"ca_cert"`
	Timeout            string
	ScriptPath         string        `mapstructure:"script_path"`
	TimeoutVal         time.Duration `mapstructure:"-"`
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
	if connInfo.Port == 0 {
		connInfo.Port = DefaultPort
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

func readPrivateKey(pk string) ([]byte, error) {
	key, _, err := pathorcontents.Read(pk)
	if err != nil {
		return nil, fmt.Errorf("Failed to read private key %q: %s", pk, err)
	}

	block, _ := pem.Decode([]byte(key))
	if block == nil {
		return nil, fmt.Errorf("Failed to read key %q: no key found", pk)
	}
	if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
		return nil, fmt.Errorf(
			"Failed to read key %q: password protected keys are\n"+
				"not supported. Please decrypt the key prior to use.", pk)
	}

	return block.Bytes, nil
}

func decryptPassword(connInfo *connectionInfo) (string, error) {
	if connInfo.Password == "" || connInfo.PasswordPrivateKey == "" {
		return connInfo.Password, nil
	}

	password, err := base64.StdEncoding.DecodeString(connInfo.Password)
	if err != nil {
		return "", err
	}

	asn1Bytes, err := readPrivateKey(connInfo.PasswordPrivateKey)
	if err != nil {
		return "", err
	}

	key, err := x509.ParsePKCS1PrivateKey(asn1Bytes)
	if err != nil {
		return "", err
	}

	out, err := rsa.DecryptPKCS1v15(nil, key, password)
	if err != nil {
		return "", err
	}

	return string(out), nil
}
