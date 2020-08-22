package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/kevinburke/ssh_config"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	E2euser = "testuser"
	E2epass = "tiger"
	E2eport = 2022
)

type o struct{}

func (o *o) Output(msg string) {
	io.Copy(os.Stdout, strings.NewReader(msg))
}

func TestSplitBastion(t *testing.T) {
	var u, h, p string
	fu := "foo"
	fh := "bar"
	fp := "2020"
	dp := "22"
	du := DefaultUser
	var cfgHasPort, cfgNoPort *ssh_config.Config
	var err error
	cfgNoPort, err = ssh_config.Decode(strings.NewReader(""))
	if err != nil {
		t.Fatalf("error parsing no-port config: %-v", err)
	}
	if u, h, p, err = splitBastion(cfgNoPort, fh); err != nil {
		t.Fatalf("error splitting bastion string: %-v", err)
	} else if u != du {
		t.Errorf("expected username to be '%s' but got '%s' instead", du, u)
	} else if h != fh {
		t.Errorf("expected hostname to be '%s' but got '%s' instead", fh, h)
	} else if p != dp {
		t.Errorf("expected port number to be '%s' but got '%s' instead", dp, p)
	}
	cfgHasPort, err = ssh_config.Decode(strings.NewReader("Port " + fp))
	if err != nil {
		t.Fatalf("error parsing port config: %-v", err)
	}
	if u, h, p, err = splitBastion(cfgHasPort, fu+"@"+fh); err != nil {
		t.Fatalf("error splitting bastion string: %-v", err)
	} else if u != fu {
		t.Errorf("expected username to be '%s' but got '%s' instead", fu, u)
	} else if h != fh {
		t.Errorf("expected hostname to be '%s' but got '%s' instead", fh, h)
	} else if p != fp {
		t.Errorf("expected port number to be '%s' but got '%s' instead", fp, p)
	}
	if u, h, p, err = splitBastion(cfgHasPort, fu+"@"+fh+":"+dp); err != nil {
		t.Fatalf("error splitting bastion string: %-v", err)
	} else if u != fu {
		t.Errorf("expected username to be '%s' but got '%s' instead", fu, u)
	} else if h != fh {
		t.Errorf("expected hostname to be '%s' but got '%s' instead", fh, h)
	} else if p != dp {
		t.Errorf("expected port number to be '%s' but got '%s' instead", dp, p)
	}
}

func TestGetSshConfigPath(t *testing.T) {
	user, err := user.Current()
	if err != nil {
		t.Fatalf("cannot get current user: %-v", err)
	}
	defpath := filepath.Join(user.HomeDir, ".ssh", "config")
	altpath := filepath.Join("foo", "bar", "config")
	if cfgpath, err := getSshConfigPath(); err != nil {
		t.Fatalf("cannot get ssh config path: %-v", err)
	} else if cfgpath != defpath {
		t.Errorf("expected config path to be '%s' but got '%s' instead", defpath, cfgpath)
	}
	os.Setenv(SSH_CONFIG_PATH, altpath)
	if cfgpath, err := getSshConfigPath(); err != nil {
		t.Fatalf("cannot get ssh config path: %-v", err)
	} else if cfgpath != altpath {
		t.Errorf("expected config path to be '%s' but got '%s' instead", altpath, cfgpath)
	}
	os.Setenv(SSH_CONFIG_PATH, "")
}

func TestNewSshConfigBstList(t *testing.T) {
	altport := "2020"
	bst1 := "bst1"
	bst2 := "foo@bst2"
	bst3 := "bar@bst3:" + altport
	target := "baz"
	bstlist := bst1 + "," + bst2 + "," + bst3
	tmpfile, err := makeTempFile("", ".tf_ssh_client_test", "ProxyJump "+bstlist)
	if tmpfile != "" {
		defer os.Remove(tmpfile)
	} else if err != nil {
		t.Fatalf("cannot create temp file: %-v", err)
	}
	os.Setenv(SSH_CONFIG_PATH, tmpfile)
	if cfg, err := newSshConfig(); err != nil {
		t.Errorf("cannot retrieve ssh config: %-v", err)
	} else if pj, err := cfg.Get(target, "ProxyJump"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if pj != bstlist {
		t.Errorf("expected bastion list to be '%s' but got '%s' instead", bstlist, pj)
	}
}

func TestNewSshConfigNestedBst(t *testing.T) {
	altport := "2020"
	bst1 := "bst1"
	bst2 := "foo@bst2"
	bst2host := "bst2"
	bst3 := "bar@bst3:" + altport
	bst3host := "bst3"
	target := "baz"
	cfgdata := `Host ` + target + `
ProxyJump ` + bst1 + `
Host ` + bst1 + `
ProxyJump ` + bst2 + `
Host ` + bst2host + `
ProxyJump ` + bst3
	tmpfile, err := makeTempFile("", ".tf_ssh_client_test", cfgdata)
	if tmpfile != "" {
		defer os.Remove(tmpfile)
	} else if err != nil {
		t.Fatalf("cannot create temp file: %-v", err)
	}
	os.Setenv(SSH_CONFIG_PATH, tmpfile)
	cfg, err := newSshConfig()
	if err != nil {
		t.Errorf("cannot retrieve ssh config: %-v", err)
	}
	if pj, err := cfg.Get(target, "ProxyJump"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if pj != bst1 {
		t.Errorf("expected bastion to be '%s' but got '%s' instead", bst1, pj)
	}
	if pj, err := cfg.Get(bst1, "ProxyJump"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if pj != bst2 {
		t.Errorf("expected bastion to be '%s' but got '%s' instead", bst2, pj)
	}
	if pj, err := cfg.Get(bst2host, "ProxyJump"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if pj != bst3 {
		t.Errorf("expected bastion to be '%s' but got '%s' instead", bst3, pj)
	}
	if pj, err := cfg.Get(bst3host, "ProxyJump"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if pj != "" {
		t.Errorf("expected bastion to be empty but got '%s' instead", pj)
	}
}

func TestNewSshConfigOther(t *testing.T) {
	altport := "2020"
	bst1 := "bst1"
	target := "baz"
	certfile := filepath.Join(os.TempDir(), "cert.crt")
	idfile := filepath.Join(os.TempDir(), "id_rsa")
	knownhostsfile := filepath.Join(os.TempDir(), "known_hosts")
	altuser := "foobar"
	cfgdata := `CertificateFile ` + certfile + `
IdentityFile ` + idfile + `
UserKnownHostsFile ` + knownhostsfile + `
User ` + DefaultUser + `
Host ` + target + `
User ` + altuser + `
ProxyJump ` + bst1 + `
Port ` + altport
	tmpfile, err := makeTempFile("", ".tf_ssh_client_test", cfgdata)
	if tmpfile != "" {
		defer os.Remove(tmpfile)
	} else if err != nil {
		t.Fatalf("cannot create temp file: %-v", err)

	}
	os.Setenv(SSH_CONFIG_PATH, tmpfile)
	cfg, err := newSshConfig()
	if err != nil {
		t.Errorf("cannot retrieve ssh config: %-v", err)
	}
	if pj, err := cfg.Get(target, "ProxyJump"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if pj != bst1 {
		t.Errorf("expected bastion to be '%s' but got '%s' instead", bst1, pj)
	}
	if value, err := cfg.Get(target, "CertificateFile"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if value != certfile {
		t.Errorf("expected certfile to be '%s' but got '%s' instead", certfile, value)
	}
	if value, err := cfg.Get(target, "IdentityFile"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if value != idfile {
		t.Errorf("expected idfile to be '%s' but got '%s' instead", idfile, value)
	}
	if value, err := cfg.Get(target, "UserKnownHostsFile"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if value != knownhostsfile {
		t.Errorf("expected knownhostsfile to be '%s' but got '%s' instead", knownhostsfile, value)
	}
	if value, err := cfg.Get(target, "User"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if value != DefaultUser {
		t.Errorf("expected user to be '%s' but got '%s' instead", DefaultUser, value)
	}
	if value, err := cfg.Get(target, "Port"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if value != altport {
		t.Errorf("expected port to be '%s' but got '%s' instead", altport, value)
	}
}

func TestNewSshConfigOtherReversed(t *testing.T) {
	altport := "2020"
	bst1 := "bst1"
	target := "baz"
	certfile := filepath.Join(os.TempDir(), "cert.crt")
	idfile := filepath.Join(os.TempDir(), "id_rsa")
	knownhostsfile := filepath.Join(os.TempDir(), "known_hosts")
	altuser := "foobar"
	cfgdata := `Host ` + target + `
User ` + altuser + `
ProxyJump ` + bst1 + `
Port ` + altport + `
Host *
CertificateFile ` + certfile + `
IdentityFile ` + idfile + `
UserKnownHostsFile ` + knownhostsfile + `
User ` + DefaultUser
	tmpfile, err := makeTempFile("", ".tf_ssh_client_test", cfgdata)
	if tmpfile != "" {
		defer os.Remove(tmpfile)
	} else if err != nil {
		t.Fatalf("cannot create temp file: %-v", err)

	}
	os.Setenv(SSH_CONFIG_PATH, tmpfile)
	cfg, err := newSshConfig()
	if err != nil {
		t.Errorf("cannot retrieve ssh config: %-v", err)
	}
	if pj, err := cfg.Get(target, "ProxyJump"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if pj != bst1 {
		t.Errorf("expected bastion to be '%s' but got '%s' instead", bst1, pj)
	}
	if value, err := cfg.Get(target, "CertificateFile"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if value != certfile {
		t.Errorf("expected certfile to be '%s' but got '%s' instead", certfile, value)
	}
	if value, err := cfg.Get(target, "IdentityFile"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if value != idfile {
		t.Errorf("expected idfile to be '%s' but got '%s' instead", idfile, value)
	}
	if value, err := cfg.Get(target, "UserKnownHostsFile"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if value != knownhostsfile {
		t.Errorf("expected knownhostsfile to be '%s' but got '%s' instead", knownhostsfile, value)
	}
	if value, err := cfg.Get(target, "User"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if value != altuser {
		t.Errorf("expected user to be '%s' but got '%s' instead", altuser, value)
	}
	if value, err := cfg.Get(target, "Port"); err != nil {
		t.Errorf("cannot retrieve ssh config value: %-v", err)
	} else if value != altport {
		t.Errorf("expected port to be '%s' but got '%s' instead", altport, value)
	}
}

func TestHostkeyReader(t *testing.T) {
	target1 := "foo"
	target2 := "bar"
	target3 := "baz"
	defport := "20"
	altport := "2020"
	hkey1 := "ssh-rsa __hostkey1__"
	hkey1alt := "ssh-rsa __hostkey1alt__"
	hkey2 := "ssh-rsa __hostkey2__"
	hkey2alt := "ssh-rsa __hostkey2alt__"
	hktest1 := target1 + " " + hkey1
	hktest2 := target1 + ":" + altport + " " + hkey1alt + " comment"
	hktest3 := target2 + ",1.2.3.4 " + hkey2
	hktest4 := target2 + ":" + altport + ",1.2.3.4 " + hkey2alt + " comment"
	hkdata := `foobar ssh-rsa __foobarhostkey__
` + hktest1 + `
` + hktest2 + `
` + hktest3 + `
` + hktest4
	hkfile, err := makeTempFile("", ".tf_ssh_client_test_hk", hkdata)
	if hkfile != "" {
		defer os.Remove(hkfile)
	} else if err != nil {
		t.Fatalf("cannot create hostkey temp file: %-v", err)
	}
	if hk, err := parseBastionHostkeyFile(target1, defport, hkfile); err != nil {
		t.Errorf("cannot parse hostkey file: %-v", err)
	} else if hk != hkey1 {
		t.Errorf("expected the hostkey to be '%s' but got '%s' instead", hkey1, hk)
	}
	if hk, err := parseBastionHostkeyFile(target1, altport, hkfile); err != nil {
		t.Errorf("cannot parse hostkey file: %-v", err)
	} else if hk != hkey1alt {
		t.Errorf("expected the hostkey to be '%s' but got '%s' instead", hkey1alt, hk)
	}
	if hk, err := parseBastionHostkeyFile(target2, defport, hkfile); err != nil {
		t.Errorf("cannot parse hostkey file: %-v", err)
	} else if hk != hkey2 {
		t.Errorf("expected the hostkey to be '%s' but got '%s' instead", hkey2, hk)
	}
	if hk, err := parseBastionHostkeyFile(target2, altport, hkfile); err != nil {
		t.Errorf("cannot parse hostkey file: %-v", err)
	} else if hk != hkey2alt {
		t.Errorf("expected the hostkey to be '%s' but got '%s' instead", hkey2alt, hk)
	}
	if hk, err := parseBastionHostkeyFile(target3, altport, hkfile); err != nil {
		t.Errorf("cannot parse hostkey file: %-v", err)
	} else if hk != "" {
		t.Errorf("expected the hostkey to be empty but got '%s' instead", hk)
	}
}

func TestHostkeyGetter(t *testing.T) {
	target := "foo"
	altport := "2020"
	hkey := "ssh-rsa __hostkey__"
	hktest := target + " " + hkey
	hkdata := `foobar ssh-rsa __foobarhostkey__
` + hktest
	hkfile, err := makeTempFile("", ".tf_ssh_client_test_hk", hkdata)
	if hkfile != "" {
		defer os.Remove(hkfile)
	} else if err != nil {
		t.Fatalf("cannot create hostkey temp file: %-v", err)
	}
	cfgDataStrict := "UserKnownHostsFile " + hkfile
	cfgDataNonStrict := "UserKnownHostsFile " + hkfile + `
StrictHostKeyChecking off`
	if cfgStrict, err := ssh_config.Decode(strings.NewReader(cfgDataStrict)); err != nil {
		t.Errorf("error parsing port config: %-v", err)
	} else if hk, err := getBastionHostkey(target, altport, cfgStrict); err != nil {
		t.Errorf("error retrieving hostkey: %-v", err)
	} else if hk != hkey {
		t.Errorf("expected the hostkey to be '%s' but got '%s' instead", hkey, hk)
	}
	if cfgNonStrict, err := ssh_config.Decode(strings.NewReader(cfgDataNonStrict)); err != nil {
		t.Errorf("error parsing port config: %-v", err)
	} else if hk, err := getBastionHostkey(target, altport, cfgNonStrict); err != nil {
		t.Errorf("error retrieving hostkey: %-v", err)
	} else if hk != "" {
		t.Errorf("expected the hostkey to be empty but got '%s' instead", hk)
	}
}

func TestMergeConnInfo(t *testing.T) {
	certdata := `--- BEGIN CERTIFICATE
__cert data__
--- END CERTIFICATE`
	iddata := "__private key__"
	target := "baz"
	user := "foo"
	hkey := "ssh-rsa __hostkey__"
	agentId := "myId"
	hostConnInfo := &connectionInfo{
		Host:          target,
		Port:          DefaultPort,
		User:          user,
		Password:      "",
		PrivateKey:    iddata,
		HostKey:       hkey,
		Certificate:   certdata,
		AgentIdentity: agentId,
	}
	bastion := "barbaz"
	bhkey := "ssh-rsa __bastion-hostkey__"
	hostAndPort := fmt.Sprintf("%s:%d", bastion, DefaultPort)
	bastionConnInfo := &connectionInfo{
		BastionHost:    bastion,
		BastionHostKey: bhkey,
		AgentIdentity:  agentId,
	}
	mergeConnInfo(bastionConnInfo, hostConnInfo)
	if bAddr := fmt.Sprintf("%s:%d", bastionConnInfo.BastionHost, bastionConnInfo.BastionPort); bAddr != hostAndPort {
		t.Errorf("expected bastion address to be '%s' but got '%s' instead", hostAndPort, bAddr)
	} else if bastionConnInfo.BastionUser != user {
		t.Errorf("expected bastion user to be '%s' but got '%s' instead", user, bastionConnInfo.BastionUser)
	} else if bastionConnInfo.BastionPassword != "" {
		t.Errorf("expected bastion password to be empty but got '%s' instead", bastionConnInfo.BastionPassword)
	} else if bastionConnInfo.BastionPrivateKey != iddata {
		t.Errorf("expected bastion id data to be '%s' but got '%s' instead", iddata, bastionConnInfo.BastionPrivateKey)
	} else if bastionConnInfo.BastionCertificate != certdata {
		t.Errorf("expected bastion cert data to be '%s' but got '%s' instead", certdata, bastionConnInfo.BastionCertificate)
	} else if bastionConnInfo.BastionHost != bastion {
		t.Errorf("expected bastion host to be '%s' but got '%s' instead", bastion, bastionConnInfo.BastionHost)
	} else if bastionConnInfo.BastionHostKey != bhkey {
		t.Errorf("expected bastion hostkey to be '%s' but got '%s' instead", bhkey, bastionConnInfo.BastionHostKey)
	} else if bastionConnInfo.BastionPort != DefaultPort {
		t.Errorf("expected bastion port to be '%d' but got '%d' instead", DefaultPort, bastionConnInfo.BastionPort)
	} else if bastionConnInfo.AgentIdentity != agentId {
		t.Errorf("expected agent identity to be '%s' but got '%s' instead", agentId, bastionConnInfo.AgentIdentity)
	}
}

func TestBuildBastionConnInfo(t *testing.T) {
	var err error
	certdata := `--- BEGIN CERTIFICATE
__cert data__
--- END CERTIFICATE`
	certfile, err := makeTempFile("", ".tf_ssh_client_test_ca", certdata)
	if certfile != "" {
		defer os.Remove(certfile)
	} else if err != nil {
		t.Fatalf("cannot create cert temp file: %-v", err)
	}
	iddata := "__private key__"
	idfile, err := makeTempFile("", ".tf_ssh_client_test_pk", iddata)
	if idfile != "" {
		defer os.Remove(idfile)
	} else if err != nil {
		t.Fatalf("cannot create id temp file: %-v", err)
	}
	target := "baz"
	hkey := "ssh-rsa __hostkey__"
	hkdata := target + " " + hkey
	knownhostsfile, err := makeTempFile("", ".tf_ssh_client_test_hk", hkdata)
	if knownhostsfile != "" {
		defer os.Remove(knownhostsfile)
	} else if err != nil {
		t.Fatalf("cannot create hostkey temp file: %-v", err)

	}
	altport := "2020"
	altportint := 2020
	bst1 := "bst1"
	altuser := "foobar"
	agentId := "myId"
	hostAndPort := target + ":" + altport
	cfgdata := `Host ` + target + `
User ` + altuser + `
ProxyJump ` + bst1 + `
Port ` + altport + `
Host *
CertificateFile ` + certfile + `
IdentityFile ` + idfile + `
UserKnownHostsFile ` + knownhostsfile + `
User ` + DefaultUser
	tmpfile, err := makeTempFile("", ".tf_ssh_client_test", cfgdata)
	if tmpfile != "" {
		defer os.Remove(tmpfile)
	} else if err != nil {
		t.Fatalf("cannot create temp file: %-v", err)

	}
	os.Setenv(SSH_CONFIG_PATH, tmpfile)
	cfg, err := newSshConfig()
	if ci, err := buildBastionConnInfo(cfg, target, &sshAgent{nil, nil, agentId}, new(connectionInfo)); err != nil {
		t.Errorf("error creating connInfo object: %-v", err)
	} else if bAddr := fmt.Sprintf("%s:%d", ci.BastionHost, ci.BastionPort); bAddr != hostAndPort {
		t.Errorf("expected bastion address to be '%s' but got '%s' instead", hostAndPort, bAddr)
	} else if ci.BastionUser != altuser {
		t.Errorf("expected bastion user to be '%s' but got '%s' instead", altuser, ci.BastionUser)
	} else if ci.BastionPassword != "" {
		t.Errorf("expected bastion password to be empty but got '%s' instead", ci.BastionPassword)
	} else if ci.BastionPrivateKey != iddata {
		t.Errorf("expected bastion id data to be '%s' but got '%s' instead", iddata, ci.BastionPrivateKey)
	} else if ci.BastionCertificate != certdata {
		t.Errorf("expected bastion cert data to be '%s' but got '%s' instead", certdata, ci.BastionCertificate)
	} else if ci.BastionHost != target {
		t.Errorf("expected bastion host to be '%s' but got '%s' instead", target, ci.BastionHost)
	} else if ci.BastionHostKey != hkey {
		t.Errorf("expected bastion hostkey to be '%s' but got '%s' instead", hkey, ci.BastionHostKey)
	} else if ci.BastionPort != altportint {
		t.Errorf("expected bastion port to be '%d' but got '%d' instead", altportint, ci.BastionPort)
	} else if ci.AgentIdentity != agentId {
		t.Errorf("expected agent identity to be '%s' but got '%s' instead", agentId, ci.AgentIdentity)
	}
}

func ExampleBastionInfo() {
	target := "foo"
	buser := "bar"
	target1 := "foobar"
	buser1 := "baz"
	c := &sshConfConn{
		Conn: nil,
		Bastions: []*sshBastion{
			&sshBastion{
				Client: nil,
				bAddr:  "",
				connInfo: &connectionInfo{
					BastionHost:        target,
					BastionUser:        buser,
					BastionPassword:    "",
					BastionPrivateKey:  "__key__",
					BastionCertificate: "__cert__",
					BastionHostKey:     "__hostkey__",
					Agent:              true,
				},
			},
			&sshBastion{
				Client: nil,
				bAddr:  "",
				connInfo: &connectionInfo{
					BastionHost:        target1,
					BastionUser:        buser1,
					BastionPassword:    "",
					BastionPrivateKey:  "__key__",
					BastionCertificate: "__cert__",
					BastionHostKey:     "__hostkey__",
					Agent:              false,
				},
			},
		},
	}
	c.BastionInfo(&_outputter{})
	// Output:
	// Using configured bastion host...
	//   Host: foo
	//   User: bar
	//   Password: false
	//   Private key: true
	//   Certificate: true
	//   SSH Agent: true
	//   Checking Host Key: true
	// Using configured bastion host...
	//   Host: foobar
	//   User: baz
	//   Password: false
	//   Private key: true
	//   Certificate: true
	//   SSH Agent: false
	//   Checking Host Key: true
}

func TestParseBastionsFromConfig(t *testing.T) {
	bhosts := []string{"b1", "b2", "b4", "b3"}
	busers := []string{"u1", "u2", "u4", "u3"}
	bastions := fmt.Sprintf("Host %s\nProxyJump %s@%s:%d\nHost 127.0.0.1\nProxyJump %s@%s:%d, %s@%s:%d , %s@%s:%d", bhosts[1], busers[2], bhosts[2], E2eport, busers[0], bhosts[0], E2eport, busers[1], bhosts[1], E2eport, busers[3], bhosts[3], E2eport)
	hostConnInfo := &connectionInfo{
		Host:     "127.0.0.1",
		Port:     E2eport,
		Password: E2epass,
		User:     E2euser,
	}
	bcc := new(sshConfConn)
	if cfg, err := ssh_config.Decode(strings.NewReader(bastions)); err != nil {
		t.Errorf("error parsing ssh config: %-v", err)
	} else if _, err := parseBastionsFromConfig(&sshAgent{nil, nil, "myId"}, hostConnInfo, nil, cfg, bcc); err != nil {
		t.Errorf("error creating the bastion list: %-v", err)
	} else if len(bcc.Bastions) != len(bhosts) {
		t.Errorf("expected '%d' numbers of bastion hosts but got '%d' instead", len(bhosts), len(bcc.Bastions))
	}
	for i := 0; i < len(bhosts); i++ {
		ci := bcc.Bastions[i].connInfo
		if ci.BastionHost != bhosts[i] {
			t.Errorf("expected b%d host to be '%s' but got '%s' instead", i, bhosts[i], ci.BastionHost)
		} else if ci.BastionUser != busers[i] {
			t.Errorf("expected b%d user to be '%s' but got '%s' instead", i, busers[i], ci.BastionUser)
		} else if ci.BastionPassword != E2epass {
			t.Errorf("expected b%d password to be '%s' but got '%s' instead", i, E2epass, ci.BastionPassword)
		}
	}
}

func TestE2e(t *testing.T) {
	connCounter := &counter{}
	ready := make(chan bool)
	bcc := new(sshConfConn)
	noBastionFound := "No bastion found."
	go startServer(connCounter, ready)
	<-ready
	bastions := fmt.Sprintf("User %s\nHost 127.0.0.1\nProxyJump localhost:%d, localhost:%d , localhost:%d", E2euser, E2eport, E2eport, E2eport)
	hostConnInfo := &connectionInfo{
		Host:     "127.0.0.1",
		Port:     E2eport,
		Password: E2epass,
		User:     E2euser,
	}
	if cfg, err := ssh_config.Decode(strings.NewReader(bastions)); err != nil {
		t.Errorf("error parsing ssh config: %-v", err)
	} else if _, err := parseBastionsFromConfig(&sshAgent{nil, nil, "myId"}, hostConnInfo, nil, cfg, bcc); err != nil {
		t.Errorf("error creating the bastion list: %-v", err)
	} else if len(bcc.Bastions) != 3 {
		t.Errorf("expected '3' bastion connInfos but got '%d' instead", len(bcc.Bastions))
	} else if _, err := sshConfigConnect("tcp", fmt.Sprintf("127.0.0.1:%d", E2eport), &sshAgent{nil, nil, "myId"}, new(sshConfConn)); err == nil {
		t.Errorf("expected a '%s' error but the func succeeded", noBastionFound)
	} else if err.Error() != noBastionFound {
		t.Errorf("expected a '%s' error but got a '%s' error instead", noBastionFound, err.Error())
	} else if conn, err := sshConfigConnect("tcp", fmt.Sprintf("127.0.0.1:%d", E2eport), &sshAgent{nil, nil, "myId"}, bcc); err != nil {
		t.Errorf("error connecting to the target: %-v", err)
	} else if connCounter.connections != 4 {
		time.Sleep(100 * time.Millisecond) // because the final connect returns immediately, so the 'server' may need a bit
		if connCounter.connections != 4 {
			t.Errorf("expected the connections counter to be '4' but got '%d' instead", connCounter.connections)
		}
	} else {
		conn.Close()
	}
}

type counter struct {
	connections int
}

type _outputter struct{}

func (o *_outputter) Output(data string) {
	fmt.Println(data)
}

func makeTempFile(dir, pfx, content string) (string, error) {
	tmpfile, err := ioutil.TempFile(dir, pfx)
	if err != nil {
		return "", err
	}

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		return tmpfile.Name(), err
	}
	if err := tmpfile.Close(); err != nil {
		return tmpfile.Name(), err
	}
	return tmpfile.Name(), nil
}

func startServer(cnt *counter, ready chan<- bool) {
	// from crypto/x/ssh example_test.go
	// authorizedKeysMap := map[string]bool{}

	// An SSH server is represented by a ServerConfig, which holds
	// certificate details and handles authentication of ServerConns.
	config := &ssh.ServerConfig{
		// Remove to disable password auth.
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Should use constant-time compare (or better, salt+hash) in
			// a production setting.
			if c.User() == E2euser && string(pass) == E2epass {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
	}

	privateKey, err := generatePrivateKey(2048)
	if err != nil {
		log.Fatal("Failed to create private key: ", err)
	}
	privateBytes := encodePrivateKeyToPEM(privateKey)

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	config.AddHostKey(private)

	// Once a ServerConfig has been configured, connections can be
	// accepted.
	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", E2eport))
	if err != nil {
		log.Fatal("failed to listen for connection: ", err)
	}
	ready <- true

	for {
		nConn, err := listener.Accept()
		if err != nil {
			log.Fatal("failed to accept incoming connection: ", err)
		}

		// count how many connections are served
		cnt.connections++

		// Before use, a handshake must be performed on the incoming
		// net.Conn.
		// conn, chans, reqs, err := ssh.NewServerConn(nConn, config)
		_, chans, reqs, err := ssh.NewServerConn(nConn, config)
		if err != nil {
			log.Fatal("failed to handshake: ", err)
		}
		// log.Printf("logged in with key %s", conn.Permissions.Extensions["pubkey-fp"])

		// The incoming Request channel must be serviced.
		go ssh.DiscardRequests(reqs)

		// service the incoming Channel channel.
		go serviceChannel(chans)
	}
}

func serviceChannel(chans <-chan ssh.NewChannel) {
	// from crypto/x/ssh tcpip.go
	// RFC 4254 7.2
	type channelOpenDirectMsg struct {
		Raddr string
		Rport uint32
		Laddr string
		Lport uint32
	}
	// Service the incoming Channel channel.
	for newChannel := range chans {
		// Channels have a type, depending on the application level
		// protocol intended. In the case of a shell, the type is
		// "session" and ServerShell may be used to present a simple
		// terminal interface.
		// We want to test port forwarding here, so we only accept "direct-tcpip" types.
		if newChannel.ChannelType() != "direct-tcpip" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}
		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Fatalf("Could not accept channel: %v", err)
		}
		// Sessions have out-of-band requests such as "shell",
		// "pty-req" and "env".  Here we handle only the
		// "shell" request.
		// In the case of direct-tcpip no request is expected.
		go func(in <-chan *ssh.Request) {
			for req := range in {
				req.Reply(false, nil)
			}
		}(requests)

		fwddata := newChannel.ExtraData()
		extra := new(channelOpenDirectMsg)
		if err := ssh.Unmarshal(fwddata, extra); err != nil {
			log.Fatalf("Could not unmarshal forward data: %v", err)
		} else if c, err := connectRemote("tcp", fmt.Sprintf("%s:%d", extra.Raddr, extra.Rport)); err != nil {
			log.Fatalf("Could not connect to remote %s:%d: %v", extra.Raddr, extra.Rport, err)
		} else {
			go io.Copy(channel, c) // channel to remote forwarding
			go io.Copy(c, channel) // remote to channel forwarding
			// return success
		}
	}
}

func connectRemote(proto, addr string) (net.Conn, error) {
	c, err := net.DialTimeout(proto, addr, 5*time.Second)
	if err != nil {
		return nil, err
	}
	return c, nil
}

// from https://gist.github.com/devinodaniel/8f9b8a4f31573f428f29ec0e884e6673
func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	// Private Key generation
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	// Validate Private Key
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	log.Println("Private Key generated")
	return privateKey, nil
}

// encodePrivateKeyToPEM encodes Private Key from RSA to PEM format
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)

	return privatePEM
}
