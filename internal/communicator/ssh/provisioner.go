package ssh

import (
	"bytes"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform/internal/communicator/shared"
	sshagent "github.com/xanzy/ssh-agent"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/knownhosts"
)

const (
	// DefaultUser is used if there is no user given
	DefaultUser = "root"

	// DefaultPort is used if there is no port given
	DefaultPort = 22

	// DefaultUnixScriptPath is used as the path to copy the file to
	// for remote execution on unix if not provided otherwise.
	DefaultUnixScriptPath = "/tmp/terraform_%RAND%.sh"
	// DefaultWindowsScriptPath is used as the path to copy the file to
	// for remote execution on windows if not provided otherwise.
	DefaultWindowsScriptPath = "C:/windows/temp/terraform_%RAND%.cmd"

	// DefaultTimeout is used if there is no timeout given
	DefaultTimeout = 5 * time.Minute

	// TargetPlatformUnix used for cleaner code, and is used if no target platform has been specified
	TargetPlatformUnix = "unix"
	//TargetPlatformWindows used for cleaner code
	TargetPlatformWindows = "windows"
)

// connectionInfo is decoded from the ConnInfo of the resource. These are the
// only keys we look at. If a PrivateKey is given, that is used instead
// of a password.
type connectionInfo struct {
	User           string
	Password       string
	PrivateKey     string
	Certificate    string
	Host           string
	HostKey        string
	Port           uint16
	Agent          bool
	ScriptPath     string
	TargetPlatform string
	Timeout        string
	TimeoutVal     time.Duration

	BastionUser        string
	BastionPassword    string
	BastionPrivateKey  string
	BastionCertificate string
	BastionHost        string
	BastionHostKey     string
	BastionPort        uint16

	AgentIdentity string
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
		case "private_key":
			connInfo.PrivateKey = v.AsString()
		case "certificate":
			connInfo.Certificate = v.AsString()
		case "host":
			connInfo.Host = v.AsString()
		case "host_key":
			connInfo.HostKey = v.AsString()
		case "port":
			if err := gocty.FromCtyValue(v, &connInfo.Port); err != nil {
				return nil, err
			}
		case "agent":
			connInfo.Agent = v.True()
		case "script_path":
			connInfo.ScriptPath = v.AsString()
		case "target_platform":
			connInfo.TargetPlatform = v.AsString()
		case "timeout":
			connInfo.Timeout = v.AsString()
		case "bastion_user":
			connInfo.BastionUser = v.AsString()
		case "bastion_password":
			connInfo.BastionPassword = v.AsString()
		case "bastion_private_key":
			connInfo.BastionPrivateKey = v.AsString()
		case "bastion_certificate":
			connInfo.BastionCertificate = v.AsString()
		case "bastion_host":
			connInfo.BastionHost = v.AsString()
		case "bastion_host_key":
			connInfo.BastionHostKey = v.AsString()
		case "bastion_port":
			if err := gocty.FromCtyValue(v, &connInfo.BastionPort); err != nil {
				return nil, err
			}
		case "agent_identity":
			connInfo.AgentIdentity = v.AsString()
		}
	}
	return connInfo, nil
}

// parseConnectionInfo is used to convert the raw configuration into the
// *connectionInfo struct.
func parseConnectionInfo(v cty.Value) (*connectionInfo, error) {
	v, err := shared.ConnectionBlockSupersetSchema.CoerceValue(v)
	if err != nil {
		return nil, err
	}

	connInfo, err := decodeConnInfo(v)
	if err != nil {
		return nil, err
	}

	// To default Agent to true, we need to check the raw string, since the
	// decoded boolean can't represent "absence of config".
	//
	// And if SSH_AUTH_SOCK is not set, there's no agent to connect to, so we
	// shouldn't try.
	agent := v.GetAttr("agent")
	if agent.IsNull() && os.Getenv("SSH_AUTH_SOCK") != "" {
		connInfo.Agent = true
	}

	if connInfo.User == "" {
		connInfo.User = DefaultUser
	}

	// Check if host is empty.
	// Otherwise return error.
	if connInfo.Host == "" {
		return nil, fmt.Errorf("host for provisioner cannot be empty")
	}

	// Format the host if needed.
	// Needed for IPv6 support.
	connInfo.Host = shared.IpFormat(connInfo.Host)

	if connInfo.Port == 0 {
		connInfo.Port = DefaultPort
	}
	// Set default targetPlatform to unix if it's empty
	if connInfo.TargetPlatform == "" {
		connInfo.TargetPlatform = TargetPlatformUnix
	} else if connInfo.TargetPlatform != TargetPlatformUnix && connInfo.TargetPlatform != TargetPlatformWindows {
		return nil, fmt.Errorf("target_platform for provisioner has to be either %s or %s", TargetPlatformUnix, TargetPlatformWindows)
	}
	// Choose an appropriate default script path based on the target platform. There is no single
	// suitable default script path which works on both UNIX and Windows targets.
	if connInfo.ScriptPath == "" && connInfo.TargetPlatform == TargetPlatformUnix {
		connInfo.ScriptPath = DefaultUnixScriptPath
	}
	if connInfo.ScriptPath == "" && connInfo.TargetPlatform == TargetPlatformWindows {
		connInfo.ScriptPath = DefaultWindowsScriptPath
	}
	if connInfo.Timeout != "" {
		connInfo.TimeoutVal = safeDuration(connInfo.Timeout, DefaultTimeout)
	} else {
		connInfo.TimeoutVal = DefaultTimeout
	}

	// Default all bastion config attrs to their non-bastion counterparts
	if connInfo.BastionHost != "" {
		// Format the bastion host if needed.
		// Needed for IPv6 support.
		connInfo.BastionHost = shared.IpFormat(connInfo.BastionHost)

		if connInfo.BastionUser == "" {
			connInfo.BastionUser = connInfo.User
		}
		if connInfo.BastionPassword == "" {
			connInfo.BastionPassword = connInfo.Password
		}
		if connInfo.BastionPrivateKey == "" {
			connInfo.BastionPrivateKey = connInfo.PrivateKey
		}
		if connInfo.BastionCertificate == "" {
			connInfo.BastionCertificate = connInfo.Certificate
		}
		if connInfo.BastionPort == 0 {
			connInfo.BastionPort = connInfo.Port
		}
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

// prepareSSHConfig is used to turn the *ConnectionInfo provided into a
// usable *SSHConfig for client initialization.
func prepareSSHConfig(connInfo *connectionInfo) (*sshConfig, error) {
	sshAgent, err := connectToAgent(connInfo)
	if err != nil {
		return nil, err
	}

	host := fmt.Sprintf("%s:%d", connInfo.Host, connInfo.Port)

	sshConf, err := buildSSHClientConfig(sshClientConfigOpts{
		user:        connInfo.User,
		host:        host,
		privateKey:  connInfo.PrivateKey,
		password:    connInfo.Password,
		hostKey:     connInfo.HostKey,
		certificate: connInfo.Certificate,
		sshAgent:    sshAgent,
	})
	if err != nil {
		return nil, err
	}

	connectFunc := ConnectFunc("tcp", host)

	var bastionConf *ssh.ClientConfig
	if connInfo.BastionHost != "" {
		bastionHost := fmt.Sprintf("%s:%d", connInfo.BastionHost, connInfo.BastionPort)

		bastionConf, err = buildSSHClientConfig(sshClientConfigOpts{
			user:        connInfo.BastionUser,
			host:        bastionHost,
			privateKey:  connInfo.BastionPrivateKey,
			password:    connInfo.BastionPassword,
			hostKey:     connInfo.HostKey,
			certificate: connInfo.BastionCertificate,
			sshAgent:    sshAgent,
		})
		if err != nil {
			return nil, err
		}

		connectFunc = BastionConnectFunc("tcp", bastionHost, bastionConf, "tcp", host)
	}

	config := &sshConfig{
		config:     sshConf,
		connection: connectFunc,
		sshAgent:   sshAgent,
	}
	return config, nil
}

type sshClientConfigOpts struct {
	privateKey  string
	password    string
	sshAgent    *sshAgent
	certificate string
	user        string
	host        string
	hostKey     string
}

func buildSSHClientConfig(opts sshClientConfigOpts) (*ssh.ClientConfig, error) {
	hkCallback := ssh.InsecureIgnoreHostKey()

	if opts.hostKey != "" {
		// The knownhosts package only takes paths to files, but terraform
		// generally wants to handle config data in-memory. Rather than making
		// the known_hosts file an exception, write out the data to a temporary
		// file to create the HostKeyCallback.
		tf, err := ioutil.TempFile("", "tf-known_hosts")
		if err != nil {
			return nil, fmt.Errorf("failed to create temp known_hosts file: %s", err)
		}
		defer tf.Close()
		defer os.RemoveAll(tf.Name())

		// we mark this as a CA as well, but the host key fallback will still
		// use it as a direct match if the remote host doesn't return a
		// certificate.
		if _, err := tf.WriteString(fmt.Sprintf("@cert-authority %s %s\n", opts.host, opts.hostKey)); err != nil {
			return nil, fmt.Errorf("failed to write temp known_hosts file: %s", err)
		}
		tf.Sync()

		hkCallback, err = knownhosts.New(tf.Name())
		if err != nil {
			return nil, err
		}
	}

	conf := &ssh.ClientConfig{
		HostKeyCallback: hkCallback,
		User:            opts.user,
	}

	if opts.privateKey != "" {
		if opts.certificate != "" {
			log.Println("using client certificate for authentication")

			certSigner, err := signCertWithPrivateKey(opts.privateKey, opts.certificate)
			if err != nil {
				return nil, err
			}
			conf.Auth = append(conf.Auth, certSigner)
		} else {
			log.Println("using private key for authentication")

			pubKeyAuth, err := readPrivateKey(opts.privateKey)
			if err != nil {
				return nil, err
			}
			conf.Auth = append(conf.Auth, pubKeyAuth)
		}
	}

	if opts.password != "" {
		conf.Auth = append(conf.Auth, ssh.Password(opts.password))
		conf.Auth = append(conf.Auth, ssh.KeyboardInteractive(
			PasswordKeyboardInteractive(opts.password)))
	}

	if opts.sshAgent != nil {
		conf.Auth = append(conf.Auth, opts.sshAgent.Auth())
	}

	return conf, nil
}

// Create a Cert Signer and return ssh.AuthMethod
func signCertWithPrivateKey(pk string, certificate string) (ssh.AuthMethod, error) {
	rawPk, err := ssh.ParseRawPrivateKey([]byte(pk))
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key %q: %s", pk, err)
	}

	pcert, _, _, _, err := ssh.ParseAuthorizedKey([]byte(certificate))
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate %q: %s", certificate, err)
	}

	usigner, err := ssh.NewSignerFromKey(rawPk)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer from raw private key %q: %s", rawPk, err)
	}

	ucertSigner, err := ssh.NewCertSigner(pcert.(*ssh.Certificate), usigner)
	if err != nil {
		return nil, fmt.Errorf("failed to create cert signer %q: %s", usigner, err)
	}

	return ssh.PublicKeys(ucertSigner), nil
}

func readPrivateKey(pk string) (ssh.AuthMethod, error) {
	// We parse the private key on our own first so that we can
	// show a nicer error if the private key has a password.
	block, _ := pem.Decode([]byte(pk))
	if block == nil {
		return nil, errors.New("Failed to read ssh private key: no key found")
	}
	if block.Headers["Proc-Type"] == "4,ENCRYPTED" {
		return nil, errors.New(
			"Failed to read ssh private key: password protected keys are\n" +
				"not supported. Please decrypt the key prior to use.")
	}

	signer, err := ssh.ParsePrivateKey([]byte(pk))
	if err != nil {
		return nil, fmt.Errorf("Failed to parse ssh private key: %s", err)
	}

	return ssh.PublicKeys(signer), nil
}

func connectToAgent(connInfo *connectionInfo) (*sshAgent, error) {
	if !connInfo.Agent {
		// No agent configured
		return nil, nil
	}

	agent, conn, err := sshagent.New()
	if err != nil {
		return nil, err
	}

	// connection close is handled over in Communicator
	return &sshAgent{
		agent: agent,
		conn:  conn,
		id:    connInfo.AgentIdentity,
	}, nil

}

// A tiny wrapper around an agent.Agent to expose the ability to close its
// associated connection on request.
type sshAgent struct {
	agent agent.Agent
	conn  net.Conn
	id    string
}

func (a *sshAgent) Close() error {
	if a.conn == nil {
		return nil
	}

	return a.conn.Close()
}

// make an attempt to either read the identity file or find a corresponding
// public key file using the typical openssh naming convention.
// This returns the public key in wire format, or nil when a key is not found.
func findIDPublicKey(id string) []byte {
	for _, d := range idKeyData(id) {
		signer, err := ssh.ParsePrivateKey(d)
		if err == nil {
			log.Println("[DEBUG] parsed id private key")
			pk := signer.PublicKey()
			return pk.Marshal()
		}

		// try it as a publicKey
		pk, err := ssh.ParsePublicKey(d)
		if err == nil {
			log.Println("[DEBUG] parsed id public key")
			return pk.Marshal()
		}

		// finally try it as an authorized key
		pk, _, _, _, err = ssh.ParseAuthorizedKey(d)
		if err == nil {
			log.Println("[DEBUG] parsed id authorized key")
			return pk.Marshal()
		}
	}

	return nil
}

// Try to read an id file using the id as the file path. Also read the .pub
// file if it exists, as the id file may be encrypted. Return only the file
// data read. We don't need to know what data came from which path, as we will
// try parsing each as a private key, a public key and an authorized key
// regardless.
func idKeyData(id string) [][]byte {
	idPath, err := filepath.Abs(id)
	if err != nil {
		return nil
	}

	var fileData [][]byte

	paths := []string{idPath}

	if !strings.HasSuffix(idPath, ".pub") {
		paths = append(paths, idPath+".pub")
	}

	for _, p := range paths {
		d, err := ioutil.ReadFile(p)
		if err != nil {
			log.Printf("[DEBUG] error reading %q: %s", p, err)
			continue
		}
		log.Printf("[DEBUG] found identity data at %q", p)
		fileData = append(fileData, d)
	}

	return fileData
}

// sortSigners moves a signer with an agent comment field matching the
// agent_identity to the head of the list when attempting authentication. This
// helps when there are more keys loaded in an agent than the host will allow
// attempts.
func (s *sshAgent) sortSigners(signers []ssh.Signer) {
	if s.id == "" || len(signers) < 2 {
		return
	}

	// if we can locate the public key, either by extracting it from the id or
	// locating the .pub file, then we can more easily determine an exact match
	idPk := findIDPublicKey(s.id)

	// if we have a signer with a connect field that matches the id, send that
	// first, otherwise put close matches at the front of the list.
	head := 0
	for i := range signers {
		pk := signers[i].PublicKey()
		k, ok := pk.(*agent.Key)
		if !ok {
			continue
		}

		// check for an exact match first
		if bytes.Equal(pk.Marshal(), idPk) || s.id == k.Comment {
			signers[0], signers[i] = signers[i], signers[0]
			break
		}

		// no exact match yet, move it to the front if it's close. The agent
		// may have loaded as a full filepath, while the config refers to it by
		// filename only.
		if strings.HasSuffix(k.Comment, s.id) {
			signers[head], signers[i] = signers[i], signers[head]
			head++
			continue
		}
	}
}

func (s *sshAgent) Signers() ([]ssh.Signer, error) {
	signers, err := s.agent.Signers()
	if err != nil {
		return nil, err
	}

	s.sortSigners(signers)
	return signers, nil
}

func (a *sshAgent) Auth() ssh.AuthMethod {
	return ssh.PublicKeysCallback(a.Signers)
}

func (a *sshAgent) ForwardToAgent(client *ssh.Client) error {
	return agent.ForwardToAgent(client, a.agent)
}
