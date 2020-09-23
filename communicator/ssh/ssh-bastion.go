package ssh

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/terraform/communicator/shared"
	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	SSH_CONFIG_ENABLER = "TF_USE_SSH_CONFIG"
	SSH_CONFIG_PATH    = "TF_SSH_CONFIG"
)

type outputter interface {
	Output(string)
}

type configGetter interface {
	Get(alias, key string) (string, error)
}
type sshConfConn struct {
	net.Conn
	Bastions []*sshBastion
}

func (c *sshConfConn) BastionInfo(o outputter) {
	for _, b := range c.Bastions {
		b.BastionInfo(o)
	}
}

type sshBastion struct {
	*ssh.Client
	bAddr    string
	connInfo *connectionInfo
}

func (c *sshConfConn) Close() error {
	if c.Conn != nil {
		c.Conn.Close()
	}
	var err error
	for i := range c.Bastions {
		i = len(c.Bastions) - 1 - i
		err = c.Bastions[i].Close()
	}
	return err
}

func (b *sshBastion) BastionInfo(o outputter) {
	o.Output(fmt.Sprintf(
		"Using configured bastion host...\n"+
			"  Host: %s\n"+
			"  User: %s\n"+
			"  Password: %t\n"+
			"  Private key: %t\n"+
			"  Certificate: %t\n"+
			"  SSH Agent: %t\n"+
			"  Checking Host Key: %t",
		b.connInfo.BastionHost, b.connInfo.BastionUser,
		b.connInfo.BastionPassword != "",
		b.connInfo.BastionPrivateKey != "",
		b.connInfo.BastionCertificate != "",
		b.connInfo.Agent,
		b.connInfo.BastionHostKey != "",
	))
}

// Parses the ssh_config file for bastion (ProxyJump) hosts and proparates bastionConfConn.
// To be used later inside sshConfigConnect().
func parseBastions(
	sshAgent *sshAgent,
	hostConnInfo *connectionInfo) (*sshConfConn, error) {
	if cfg, err := newSshConfig(); err != nil {
		return nil, err
	} else {
		return parseBastionsFromConfig(sshAgent, hostConnInfo, nil, cfg, new(sshConfConn))
	}

}

// The actual "parser" function.
func parseBastionsFromConfig(
	sshAgent *sshAgent,
	hostConnInfo *connectionInfo,
	bastionConnInfo *connectionInfo,
	cfg configGetter,
	bastionConfConn *sshConfConn) (*sshConfConn, error) {
	host := hostConnInfo.Host
	port := hostConnInfo.Port
	if bastionConnInfo != nil {
		host = bastionConnInfo.BastionHost
		port = bastionConnInfo.BastionPort
	}
	addr := fmt.Sprintf("%s:%d", host, port)
	if bastions, err := cfg.Get(host, "ProxyJump"); err != nil {
		return nil, err
	} else if bastions != "" {
		for _, b := range strings.Split(bastions, ",") {
			b = strings.Trim(b, " ")
			if newConnInfo, err := buildBastionConnInfo(cfg, b, sshAgent, hostConnInfo); err != nil {
				return nil, err
			} else {
				bastionConfConn.Bastions = append(bastionConfConn.Bastions, &sshBastion{
					bAddr:    addr,
					connInfo: newConnInfo})
				if _, err := parseBastionsFromConfig(sshAgent, hostConnInfo, newConnInfo, cfg, bastionConfConn); err != nil {
					return nil, err
				}
			}
		}
	}
	return bastionConfConn, nil
}

func newSshConfig() (configGetter, error) {
	sshConfigPath, err := getSshConfigPath()
	if err != nil {
		return nil, err
	}
	if sshConfigReader, err := os.Open(sshConfigPath); err != nil {
		return nil, err
	} else {
		return ssh_config.Decode(sshConfigReader)
	}
}

func getSshConfigPath() (string, error) {
	sshConfigPath := os.Getenv(SSH_CONFIG_PATH)
	if sshConfigPath == "" {
		// the parser used currently has an incosistent API regarding the returned objects
		// for file vs. string parsing
		// therefore we try to find the user's home ourselves
		// return ssh_config.DefaultUserSettings, nil
		if homedir, err := getUserHome(); err != nil {
			return "", err
		} else {
			sshConfigPath = filepath.Join(homedir, ".ssh", "config")
		}
	}
	return sshConfigPath, nil
}

// SshConfigConnectFunc is a convenience method for returning a function
// that connects to a host over a potential series of bastion connections
// acc. to what is defined in the ssh_conf file.
// parseBastions() must be called first in order to propagate the bastion hosts.
func SshConfigConnectFunc(
	proto, addr string,
	sshAgent *sshAgent,
	bastionConfConn *sshConfConn) func() (net.Conn, error) {
	return func() (net.Conn, error) {
		return sshConfigConnect(proto, addr, sshAgent, bastionConfConn)
	}
}

func sshConfigConnect(
	proto, addr string,
	sshAgent *sshAgent,
	bastionConfConn *sshConfConn) (net.Conn, error) {
	if len(bastionConfConn.Bastions) == 0 {
		return nil, errors.New("No bastion found.")
	}
	bastion := bastionConfConn.Bastions[0]
	sshClientConf, err := buildBastionSSHClientConfig(bastion.connInfo, sshAgent)
	if err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] Connecting to bastion: %s", addr)
	bastionClient, err := ssh.Dial(proto, addr, sshClientConf)
	if err != nil {
		return nil, fmt.Errorf("Error connecting to bastion: %s", err)
	}
	bastion.Client = bastionClient
	for i := 1; i < len(bastionConfConn.Bastions); i++ {
		bastionHop := bastionConfConn.Bastions[i-1]
		bastion := bastionConfConn.Bastions[i]
		sshClientConf, err := buildBastionSSHClientConfig(bastion.connInfo, sshAgent)
		if err != nil {
			return nil, err
		}
		if bastionClient, err := connectNestedBastion(bastionHop, proto, bastion.bAddr, sshClientConf); err != nil {
			return nil, err
		} else {
			bastion.Client = bastionClient
		}

	}
	bastion = bastionConfConn.Bastions[len(bastionConfConn.Bastions)-1]
	log.Printf("[DEBUG] Connecting via bastion (%s) to host: %s", bastion.bAddr, addr)
	conn, err := bastion.Dial(proto, addr)
	if err != nil {
		bastionConfConn.Close()
		return nil, err
	}
	bastionConfConn.Conn = conn
	return bastionConfConn, nil
}

func buildBastionConnInfo(
	cfg configGetter,
	b string,
	sshAgent *sshAgent,
	hostConnInfo *connectionInfo) (*connectionInfo, error) {
	var bPrivKey, bHostKey, bCert string
	var bPortInt int64
	bUser, bHost, bPort, err := splitBastion(cfg, b)
	if err != nil {
		return nil, err
	}
	bPortInt, err = strconv.ParseInt(bPort, 10, 0)
	if err != nil {
		return nil, err
	}
	if bUser == "" {
		if bUser, err = cfg.Get(bHost, "User"); err != nil {
			return nil, err
		}
	}
	if bPrivKey, err = catBastionConfigFile(cfg.Get(bHost, "IdentityFile")); err != nil {
		return nil, err
	}
	if bHostKey, err = getBastionHostkey(bHost, bPort, cfg); err != nil {
		return nil, err
	}
	if bCert, err = catBastionConfigFile(cfg.Get(bHost, "CertificateFile")); err != nil {
		return nil, err
	}
	bConnInfo := &connectionInfo{
		BastionUser:        bUser,
		BastionPassword:    "", // connInfo.Password, # not supported by ssh_config
		BastionPrivateKey:  bPrivKey,
		BastionCertificate: bCert,
		BastionHost:        bHost,
		BastionHostKey:     bHostKey,
		BastionPort:        int(bPortInt),
		AgentIdentity:      sshAgent.id,
	}
	mergeConnInfo(bConnInfo, hostConnInfo)
	return bConnInfo, nil
}

func buildBastionSSHClientConfig(
	connInfo *connectionInfo,
	sshAgent *sshAgent) (*ssh.ClientConfig, error) {
	if bConf, err := buildSSHClientConfig(sshClientConfigOpts{
		user:        connInfo.BastionUser,
		host:        connInfo.BastionHost,
		privateKey:  connInfo.BastionPrivateKey,
		password:    connInfo.BastionPassword,
		hostKey:     connInfo.BastionHostKey,
		certificate: connInfo.BastionCertificate,
		sshAgent:    sshAgent,
	}); err != nil {
		return nil, err
	} else {
		return bConf, nil
	}
}

func connectNestedBastion(
	bastion *sshBastion,
	proto, addr string,
	sshClientConf *ssh.ClientConfig) (*ssh.Client, error) {
	log.Printf("[DEBUG] Connecting via bastion (%s) to nested bastion: %s", bastion.bAddr, addr)
	conn, err := bastion.Dial(proto, addr)
	if err != nil {
		bastion.Close()
		return nil, err
	}
	log.Printf("[DEBUG] Connection to bastion established. Handshaking for user %v", sshClientConf.User)
	sshConn, sshChan, req, err := ssh.NewClientConn(conn, addr, sshClientConf)
	if err != nil {
		err = errwrap.Wrapf(fmt.Sprintf("SSH authentication failed (%s@%s): {{err}}", sshClientConf.User, addr), err)

		return nil, err
	}
	return ssh.NewClient(sshConn, sshChan, req), nil
}

func splitBastion(
	cfg configGetter,
	b string) (string, string, string, error) {
	bUser := ""
	bHost := ""
	bPort := ""
	var err error
	if strings.Index(b, "@") > -1 {
		bUser = strings.Split(b, "@")[0]
		bHost = strings.Split(b, "@")[1]
	} else {
		if bUser, err = cfg.Get(strings.Split(b, ":")[0], "User"); err != nil {
			return "", "", "", err
		} else if bUser == "" {
			bUser = DefaultUser
		}
		bHost = b
	}
	if strings.Index(bHost, ":") > -1 {
		bPort = strings.Split(bHost, ":")[1]
		bHost = strings.Split(bHost, ":")[0]
	} else {
		if bPort, err = cfg.Get(bHost, "Port"); err != nil {
			return "", "", "", err
		} else if bPort == "" {
			bPort = fmt.Sprintf("%d", DefaultPort)
		}
	}
	return bUser, bHost, bPort, nil
}

func getBastionHostkey(
	bHost, bPort string,
	cfg configGetter) (string, error) {
	if strictHk, err := cfg.Get(bHost, "StrictHostKeyChecking"); err != nil {
		return "", err
	} else if strictHk == "no" || strictHk == "off" {
		return "", nil
	} else if keyfile, err := cfg.Get(bHost, "UserKnownHostsFile"); err != nil {
		return "", err
	} else if keyfile == "" {
		return "", nil
	} else {
		return parseBastionHostkeyFile(bHost, bPort, keyfile)
	}
}

func parseBastionHostkeyFile(bHost, bPort, keyfile string) (string, error) {
	keys, err := catBastionConfigFile(keyfile, nil)
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(strings.NewReader(keys))
	bHostPort := fmt.Sprintf("%s:%s", bHost, bPort)
	bKey := ""
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), " ")
		hostpfx := strings.Split(parts[0], ",")[0] // trim potential IP
		candidate := strings.Join(parts[1:3], " ") // trim potential comment
		if hostpfx == bHostPort {
			return candidate, nil
		} else if hostpfx == bHost {
			bKey = candidate
		}
	}
	return bKey, nil
}

func mergeConnInfo(dst, src *connectionInfo) {
	// Format the bastion host if needed.
	// Needed for IPv6 support.
	dst.BastionHost = shared.IpFormat(dst.BastionHost)

	if dst.BastionUser == "" {
		dst.BastionUser = src.User
	}
	if dst.BastionPassword == "" {
		dst.BastionPassword = src.Password
	}
	if dst.BastionPrivateKey == "" {
		dst.BastionPrivateKey = src.PrivateKey
	}
	if dst.BastionCertificate == "" {
		dst.BastionCertificate = src.Certificate
	}
	if dst.BastionPort == 0 {
		dst.BastionPort = src.Port
	}
}

func catBastionConfigFile(
	conffile string,
	_err error) (string, error) {
	if _err != nil {
		return "", _err
	}
	if conffile == "" {
		return "", nil
	}
	if strings.Index(conffile, "~/") == 0 {
		if home, err := getUserHome(); err != nil {
			return "", err
		} else {
			conffile = home + conffile[1:]
		}
	}
	if data, err := ioutil.ReadFile(conffile); err != nil {
		return "", err
	} else {
		return string(data), nil
	}
}

func getUserHome() (string, error) {
	if user, err := user.Current(); err != nil {
		return "", err
	} else {
		return user.HomeDir, nil
	}
}
