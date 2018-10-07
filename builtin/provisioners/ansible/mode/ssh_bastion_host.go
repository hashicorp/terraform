package mode

import (
	"fmt"
	"time"

	"golang.org/x/crypto/ssh"
)

type bastionHost struct {
	connInfo *connectionInfo
}

func newBastionHostFromConnectionInfo(connInfo *connectionInfo) *bastionHost {
	return &bastionHost{
		connInfo: connInfo,
	}
}

func (v *bastionHost) agent() bool {
	return v.connInfo.Agent
}

func (v *bastionHost) inUse() bool {
	return v.connInfo.BastionHost != ""
}

func (v *bastionHost) host() string {
	return v.connInfo.BastionHost
}

func (v *bastionHost) port() int {
	return v.connInfo.BastionPort
}

func (v *bastionHost) user() string {
	return v.connInfo.BastionUser
}

func (v *bastionHost) pemFile() string {
	return v.connInfo.BastionPrivateKey
}

func (v *bastionHost) hostKey() string {
	return v.connInfo.BastionHostKey
}

func (v *bastionHost) timeout() time.Duration {
	return v.connInfo.TimeoutVal
}

func (v *bastionHost) receiveHostKey(hostKey string) {
	v.connInfo.BastionHostKey = hostKey
}

func (v *bastionHost) connect() (*ssh.Client, error) {
	configurator := &sshConfigurator{
		provider: v,
	}
	sshConfig, err := configurator.sshConfig()
	if err != nil {
		return nil, err
	}
	return ssh.Dial("tcp", fmt.Sprintf("%s:%d", v.host(), v.port()), sshConfig)
}
