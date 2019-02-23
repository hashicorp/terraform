// +build !race

package ssh

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
	"golang.org/x/crypto/ssh"
)

// private key for mock server
const testServerPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA19lGVsTqIT5iiNYRgnoY1CwkbETW5cq+Rzk5v/kTlf31XpSU
70HVWkbTERECjaYdXM2gGcbb+sxpq6GtXf1M3kVomycqhxwhPv4Cr6Xp4WT/jkFx
9z+FFzpeodGJWjOH6L2H5uX1Cvr9EDdQp9t9/J32/qBFntY8GwoUI/y/1MSTmMiF
tupdMODN064vd3gyMKTwrlQ8tZM6aYuyOPsutLlUY7M5x5FwMDYvnPDSeyT/Iw0z
s3B+NCyqeeMd2T7YzQFnRATj0M7rM5LoSs7DVqVriOEABssFyLj31PboaoLhOKgc
qoM9khkNzr7FHVvi+DhYM2jD0DwvqZLN6NmnLwIDAQABAoIBAQCGVj+kuSFOV1lT
+IclQYA6bM6uY5mroqcSBNegVxCNhWU03BxlW//BE9tA/+kq53vWylMeN9mpGZea
riEMIh25KFGWXqXlOOioH8bkMsqA8S7sBmc7jljyv+0toQ9vCCtJ+sueNPhxQQxH
D2YvUjfzBQ04I9+wn30BByDJ1QA/FoPsunxIOUCcRBE/7jxuLYcpR+JvEF68yYIh
atXRld4W4in7T65YDR8jK1Uj9XAcNeDYNpT/M6oFLx1aPIlkG86aCWRO19S1jLPT
b1ZAKHHxPMCVkSYW0RqvIgLXQOR62D0Zne6/2wtzJkk5UCjkSQ2z7ZzJpMkWgDgN
ifCULFPBAoGBAPoMZ5q1w+zB+knXUD33n1J+niN6TZHJulpf2w5zsW+m2K6Zn62M
MXndXlVAHtk6p02q9kxHdgov34Uo8VpuNjbS1+abGFTI8NZgFo+bsDxJdItemwC4
KJ7L1iz39hRN/ZylMRLz5uTYRGddCkeIHhiG2h7zohH/MaYzUacXEEy3AoGBANz8
e/msleB+iXC0cXKwds26N4hyMdAFE5qAqJXvV3S2W8JZnmU+sS7vPAWMYPlERPk1
D8Q2eXqdPIkAWBhrx4RxD7rNc5qFNcQWEhCIxC9fccluH1y5g2M+4jpMX2CT8Uv+
3z+NoJ5uDTXZTnLCfoZzgZ4nCZVZ+6iU5U1+YXFJAoGBANLPpIV920n/nJmmquMj
orI1R/QXR9Cy56cMC65agezlGOfTYxk5Cfl5Ve+/2IJCfgzwJyjWUsFx7RviEeGw
64o7JoUom1HX+5xxdHPsyZ96OoTJ5RqtKKoApnhRMamau0fWydH1yeOEJd+TRHhc
XStGfhz8QNa1dVFvENczja1vAoGABGWhsd4VPVpHMc7lUvrf4kgKQtTC2PjA4xoc
QJ96hf/642sVE76jl+N6tkGMzGjnVm4P2j+bOy1VvwQavKGoXqJBRd5Apppv727g
/SM7hBXKFc/zH80xKBBgP/i1DR7kdjakCoeu4ngeGywvu2jTS6mQsqzkK+yWbUxJ
I7mYBsECgYB/KNXlTEpXtz/kwWCHFSYA8U74l7zZbVD8ul0e56JDK+lLcJ0tJffk
gqnBycHj6AhEycjda75cs+0zybZvN4x65KZHOGW/O/7OAWEcZP5TPb3zf9ned3Hl
NsZoFj52ponUM6+99A2CmezFCN16c4mbA//luWF+k3VVqR6BpkrhKw==
-----END RSA PRIVATE KEY-----`

// this cert was signed by the key from testCAPublicKey
const testServerHostCert = `ssh-rsa-cert-v01@openssh.com AAAAHHNzaC1yc2EtY2VydC12MDFAb3BlbnNzaC5jb20AAAAgvQ3Bs1ex7277b9q6I0fNaWsVEC16f+LcT8RLPSVMEVMAAAADAQABAAABAQDX2UZWxOohPmKI1hGCehjULCRsRNblyr5HOTm/+ROV/fVelJTvQdVaRtMREQKNph1czaAZxtv6zGmroa1d/UzeRWibJyqHHCE+/gKvpenhZP+OQXH3P4UXOl6h0YlaM4fovYfm5fUK+v0QN1Cn2338nfb+oEWe1jwbChQj/L/UxJOYyIW26l0w4M3Tri93eDIwpPCuVDy1kzppi7I4+y60uVRjsznHkXAwNi+c8NJ7JP8jDTOzcH40LKp54x3ZPtjNAWdEBOPQzuszkuhKzsNWpWuI4QAGywXIuPfU9uhqguE4qByqgz2SGQ3OvsUdW+L4OFgzaMPQPC+pks3o2acvAAAAAAAAAAAAAAACAAAAB2NhLXRlc3QAAAANAAAACTEyNy4wLjAuMQAAAABag0jkAAAAAHDcHtAAAAAAAAAAAAAAAAAAAAEXAAAAB3NzaC1yc2EAAAADAQABAAABAQCrozyZIhdEvalCn+eSzHH94cO9ykiywA13ntWI7mJcHBwYTeCYWG8E9zGXyp2iDOjCGudM0Tdt8o0OofKChk9Z/qiUN0G8y1kmaXBlBM3qA5R9NPpvMYMNkYLfX6ivtZCnqrsbzaoqN2Oc/7H2StHzJWh/XCGu9otQZA6vdv1oSmAsZOjw/xIGaGQqDUaLq21J280PP1qSbdJHf76iSHE+TWe3YpqV946JWM5tCh0DykZ10VznvxYpUjzhr07IN3tVKxOXbPnnU7lX6IaLIWgfzLqwSyheeux05c3JLF9iF4sFu8ou4hwQz1iuUTU1jxgwZP0w/bkXgFFs0949lW81AAABDwAAAAdzc2gtcnNhAAABAEyoiVkZ5z79nh3WSU5mU2U7e2BItnnEqsJIm9EN+35uG0yORSXmQoaa9mtli7G3r79tyqEJd/C95EdNvU/9TjaoDcbH8OHP+Ue9XSfUzBuQ6bGSXe6mlZlO7QJ1cIyWphFP3MkrweDSiJ+SpeXzLzZkiJ7zKv5czhBEyG/MujFgvikotL+eUNG42y2cgsesXSjENSBS3l11q55a+RM2QKt3W32im8CsSxrH6Mz6p4JXQNgsVvZRknLxNlWXULFB2HLTunPKzJNMTf6xZf66oivSBAXVIdNKhlVpAQ3dT/dW5K6J4aQF/hjWByyLprFwZ16cPDqvtalnTCpbRYelNbw=`

const testCAPublicKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCrozyZIhdEvalCn+eSzHH94cO9ykiywA13ntWI7mJcHBwYTeCYWG8E9zGXyp2iDOjCGudM0Tdt8o0OofKChk9Z/qiUN0G8y1kmaXBlBM3qA5R9NPpvMYMNkYLfX6ivtZCnqrsbzaoqN2Oc/7H2StHzJWh/XCGu9otQZA6vdv1oSmAsZOjw/xIGaGQqDUaLq21J280PP1qSbdJHf76iSHE+TWe3YpqV946JWM5tCh0DykZ10VznvxYpUjzhr07IN3tVKxOXbPnnU7lX6IaLIWgfzLqwSyheeux05c3JLF9iF4sFu8ou4hwQz1iuUTU1jxgwZP0w/bkXgFFs0949lW81`

func newMockLineServer(t *testing.T, signer ssh.Signer, pubKey string) string {
	serverConfig := &ssh.ServerConfig{
		PasswordCallback:  acceptUserPass("user", "pass"),
		PublicKeyCallback: acceptPublicKey(pubKey),
	}

	var err error
	if signer == nil {
		signer, err = ssh.ParsePrivateKey([]byte(testServerPrivateKey))
		if err != nil {
			t.Fatalf("unable to parse private key: %s", err)
		}
	}
	serverConfig.AddHostKey(signer)

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Unable to listen for connection: %s", err)
	}

	go func() {
		defer l.Close()
		c, err := l.Accept()
		if err != nil {
			t.Errorf("Unable to accept incoming connection: %s", err)
		}
		defer c.Close()
		conn, chans, _, err := ssh.NewServerConn(c, serverConfig)
		if err != nil {
			t.Logf("Handshaking error: %v", err)
		}
		t.Log("Accepted SSH connection")

		for newChannel := range chans {
			channel, requests, err := newChannel.Accept()
			if err != nil {
				t.Errorf("Unable to accept channel.")
			}
			t.Log("Accepted channel")

			go func(in <-chan *ssh.Request) {
				for req := range in {
					if req.WantReply {
						req.Reply(true, nil)
					}
				}
			}(requests)

			go func(newChannel ssh.NewChannel) {
				conn.OpenChannel(newChannel.ChannelType(), nil)
			}(newChannel)

			defer channel.Close()
		}
		conn.Close()
	}()

	return l.Addr().String()
}

func TestNew_Invalid(t *testing.T) {
	address := newMockLineServer(t, nil, testClientPublicKey)
	parts := strings.Split(address, ":")

	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":     "ssh",
				"user":     "user",
				"password": "i-am-invalid",
				"host":     parts[0],
				"port":     parts[1],
				"timeout":  "30s",
			},
		},
	}

	c, err := New(r)
	if err != nil {
		t.Fatalf("error creating communicator: %s", err)
	}

	err = c.Connect(nil)
	if err == nil {
		t.Fatal("should have had an error connecting")
	}
}

func TestStart(t *testing.T) {
	address := newMockLineServer(t, nil, testClientPublicKey)
	parts := strings.Split(address, ":")

	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":     "ssh",
				"user":     "user",
				"password": "pass",
				"host":     parts[0],
				"port":     parts[1],
				"timeout":  "30s",
			},
		},
	}

	c, err := New(r)
	if err != nil {
		t.Fatalf("error creating communicator: %s", err)
	}

	var cmd remote.Cmd
	stdout := new(bytes.Buffer)
	cmd.Command = "echo foo"
	cmd.Stdout = stdout

	err = c.Start(&cmd)
	if err != nil {
		t.Fatalf("error executing remote command: %s", err)
	}
}

// TestKeepAlives verifies that the keepalive messages don't interfere with
// normal operation of the client.
func TestKeepAlives(t *testing.T) {
	address := newMockLineServer(t, nil, testClientPublicKey)
	parts := strings.Split(address, ":")

	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":     "ssh",
				"user":     "user",
				"password": "pass",
				"host":     parts[0],
				"port":     parts[1],
				"timeout":  "30s",
			},
		},
	}

	c, err := New(r)
	if err != nil {
		t.Fatalf("error creating communicator: %s", err)
	}

	if err := c.Connect(nil); err != nil {
		t.Fatal(err)
	}

	var cmd remote.Cmd
	stdout := new(bytes.Buffer)
	cmd.Command = "echo foo"
	cmd.Stdout = stdout

	// wait a bit before executing the command, so that at least 1 keepalive is sent
	time.Sleep(3 * time.Second)

	err = c.Start(&cmd)
	if err != nil {
		t.Fatalf("error executing remote command: %s", err)
	}
}

func TestLostConnection(t *testing.T) {
	address := newMockLineServer(t, nil, testClientPublicKey)
	parts := strings.Split(address, ":")

	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":     "ssh",
				"user":     "user",
				"password": "pass",
				"host":     parts[0],
				"port":     parts[1],
				"timeout":  "30s",
			},
		},
	}

	c, err := New(r)
	if err != nil {
		t.Fatalf("error creating communicator: %s", err)
	}

	var cmd remote.Cmd
	stdout := new(bytes.Buffer)
	cmd.Command = "echo foo"
	cmd.Stdout = stdout

	err = c.Start(&cmd)
	if err != nil {
		t.Fatalf("error executing remote command: %s", err)
	}

	// The test server can't execute anything, so Wait will block, unless
	// there's an error.  Disconnect the communicator transport, to cause the
	// command to fail.
	go func() {
		time.Sleep(100 * time.Millisecond)
		c.Disconnect()
	}()

	err = cmd.Wait()
	if err == nil {
		t.Fatal("expected communicator error")
	}
}

func TestHostKey(t *testing.T) {
	// get the server's public key
	signer, err := ssh.ParsePrivateKey([]byte(testServerPrivateKey))
	if err != nil {
		t.Fatalf("unable to parse private key: %v", err)
	}
	pubKey := fmt.Sprintf("ssh-rsa %s", base64.StdEncoding.EncodeToString(signer.PublicKey().Marshal()))

	address := newMockLineServer(t, nil, testClientPublicKey)
	host, p, _ := net.SplitHostPort(address)
	port, _ := strconv.Atoi(p)

	connInfo := &connectionInfo{
		User:     "user",
		Password: "pass",
		Host:     host,
		HostKey:  pubKey,
		Port:     port,
		Timeout:  "30s",
	}

	cfg, err := prepareSSHConfig(connInfo)
	if err != nil {
		t.Fatal(err)
	}

	c := &Communicator{
		connInfo: connInfo,
		config:   cfg,
	}

	var cmd remote.Cmd
	stdout := new(bytes.Buffer)
	cmd.Command = "echo foo"
	cmd.Stdout = stdout

	if err := c.Start(&cmd); err != nil {
		t.Fatal(err)
	}
	if err := c.Disconnect(); err != nil {
		t.Fatal(err)
	}

	// now check with the wrong HostKey
	address = newMockLineServer(t, nil, testClientPublicKey)
	_, p, _ = net.SplitHostPort(address)
	port, _ = strconv.Atoi(p)

	connInfo.HostKey = testClientPublicKey
	connInfo.Port = port

	cfg, err = prepareSSHConfig(connInfo)
	if err != nil {
		t.Fatal(err)
	}

	c = &Communicator{
		connInfo: connInfo,
		config:   cfg,
	}

	err = c.Start(&cmd)
	if err == nil || !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("expected host key mismatch, got error:%v", err)
	}
}

func TestHostCert(t *testing.T) {
	pk, _, _, _, err := ssh.ParseAuthorizedKey([]byte(testServerHostCert))
	if err != nil {
		t.Fatal(err)
	}

	signer, err := ssh.ParsePrivateKey([]byte(testServerPrivateKey))
	if err != nil {
		t.Fatal(err)
	}

	signer, err = ssh.NewCertSigner(pk.(*ssh.Certificate), signer)
	if err != nil {
		t.Fatal(err)
	}

	address := newMockLineServer(t, signer, testClientPublicKey)
	host, p, _ := net.SplitHostPort(address)
	port, _ := strconv.Atoi(p)

	connInfo := &connectionInfo{
		User:     "user",
		Password: "pass",
		Host:     host,
		HostKey:  testCAPublicKey,
		Port:     port,
		Timeout:  "30s",
	}

	cfg, err := prepareSSHConfig(connInfo)
	if err != nil {
		t.Fatal(err)
	}

	c := &Communicator{
		connInfo: connInfo,
		config:   cfg,
	}

	var cmd remote.Cmd
	stdout := new(bytes.Buffer)
	cmd.Command = "echo foo"
	cmd.Stdout = stdout

	if err := c.Start(&cmd); err != nil {
		t.Fatal(err)
	}
	if err := c.Disconnect(); err != nil {
		t.Fatal(err)
	}

	// now check with the wrong HostKey
	address = newMockLineServer(t, signer, testClientPublicKey)
	_, p, _ = net.SplitHostPort(address)
	port, _ = strconv.Atoi(p)

	connInfo.HostKey = testClientPublicKey
	connInfo.Port = port

	cfg, err = prepareSSHConfig(connInfo)
	if err != nil {
		t.Fatal(err)
	}

	c = &Communicator{
		connInfo: connInfo,
		config:   cfg,
	}

	err = c.Start(&cmd)
	if err == nil || !strings.Contains(err.Error(), "authorities") {
		t.Fatalf("expected host key mismatch, got error:%v", err)
	}
}

const SERVER_PEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA8CkDr7uxCFt6lQUVwS8NyPO+fQNxORoGnMnN/XhVJZvpqyKR
Uji9R0d8D66bYxUUsabXjP2y4HTVzbZtnvXFZZshk0cOtJjjekpYJaLK2esPR/iX
wvSltNkrDQDPN/RmgEEMIevW8AgrPsqrnybFHxTpd7rEUHXBOe4nMNRIg3XHykB6
jZk8q5bBPUe3I/f0DK5TJEBpTc6dO3P/j93u55VUqr39/SPRHnld2mCw+c8v6UOh
sssO/DIZFPScD3DYqsk2N+/nz9zXfcOTdWGhawgxuIo1DTokrNQbG3pDrLqcWgqj
13vqJFCmRA0O2CQIwJePd6+Np/XO3Uh/KL6FlQIDAQABAoIBAQCmvQMXNmvCDqk7
30zsVDvw4fHGH+azK3Od1aqTqcEMHISOUbCtckFPxLzIsoSltRQqB1kuRVG07skm
Stsu+xny4lLcSwBVuLRuykEK2EyYIc/5Owo6y9pkhkaSf5ZfFes4bnD6+B/BhRpp
PRMMq0E+xCkX/G6iIi9mhgdlqm0x/vKtjzQeeshw9+gRcRLUpX+UeKFKXMXcDayx
qekr1bAaQKNBhTK+CbZjcqzG4f+BXVGRTZ9nsPAV+yTnWUCU0TghwPmtthHbebqa
9hlkum7qik/bQj/tjJ8/b0vTfHQSVxhtPG/ZV2Tn9ZuL/vrkYqeyMU8XkJ/uaEvH
WPyOcB4BAoGBAP5o5JSEtPog+U3JFrLNSRjz5ofZNVkJzice+0XyqlzJDHhX5tF8
mriYQZLLXYhckBm4IdkhTn/dVbXNQTzyy2WVuO5nU8bkCMvGL9CGpW4YGqwGf7NX
e4H3emtRjLv8VZpUHe/RUUDhmYvMSt1qmXuskfpROuGfLhQBUd6A4J+BAoGBAPGp
UcMKjrxZ5qjYU6DLgS+xeca4Eu70HgdbSQbRo45WubXjyXvTRFij36DrpxJWf1D7
lIsyBifoTra/lAuC1NQXGYWjTCdk2ey8Ll5qOgiXvE6lINHABr+U/Z90/g6LuML2
VzaZbq/QLcT3yVsdyTogKckzCaKsCpusyHE1CXAVAoGAd6kMglKc8N0bhZukgnsN
+5+UeacPcY6sGTh4RWErAjNKGzx1A2lROKvcg9gFaULoQECcIw2IZ5nKW5VsLueg
BWrTrcaJ4A2XmYjhKnp6SvspaGoyHD90hx/Iw7t6r1yzQsB3yDmytwqldtyjBdvC
zynPC2azhDWjraMlR7tka4ECgYAxwvLiHa9sm3qCtCDsUFtmrb3srITBjaUNUL/F
1q8+JR+Sk7gudj9xnTT0VvINNaB71YIt83wPBagHu4VJpYQbtDH+MbUBu6OgOtO1
f1w53rzY2OncJxV8p7pd9mJGLoE6LC2jQY7oRw7Vq0xcJdME1BCmrIrEY3a/vaF8
pjYuTQKBgQCIOH23Xita8KmhH0NdlWxZfcQt1j3AnOcKe6UyN4BsF8hqS7eTA52s
WjG5X2IBl7gs1eMM1qkqR8npS9nwfO/pBmZPwjiZoilypXxWj+c+P3vwre2yija4
bXgFVj4KFBwhr1+8KcobxC0SAPEouMvSkxzjjw+gnebozUtPlud9jA==
-----END RSA PRIVATE KEY-----
`
const CLIENT_CERT_SIGNED_BY_SERVER = `ssh-rsa-cert-v01@openssh.com AAAAHHNzaC1yc2EtY2VydC12MDFAb3BlbnNzaC5jb20AAAAgbMDNUn4M2TtzrSH7MOT2QsvLzZWjehJ5TYrBOp9p+lwAAAADAQABAAABAQCyu57E7zIWRyEWuaiOiikOSZKFjbwLkpE9fboFfLLsNUJj4zw+5bZUJtzWK8roPjgL8s1oPncro5wuTtI2Nu4fkpeFK0Hb33o6Eyksuj4Om4+6Uemn1QEcb0bZqK8Zyg9Dg9deP7LeE0v78b5/jZafFgwxv+/sMhM0PRD34NCDYcYmkkHlvQtQWFAdbPXCgghObedZyYdoqZVuhTsiPMWtQS/cc9M4tv6mPOuQlhZt3R/Oh/kwUyu45oGRb5bhO4JicozFS3oeClpU+UMbgslkzApJqxZBWN7+PDFSZhKk2GslyeyP4sH3E30Z00yVi/lQYgmQsB+Hg6ClemNQMNu/AAAAAAAAAAAAAAACAAAABHVzZXIAAAAIAAAABHVzZXIAAAAAWzBjXAAAAAB/POfPAAAAAAAAAAAAAAAAAAABFwAAAAdzc2gtcnNhAAAAAwEAAQAAAQEA8CkDr7uxCFt6lQUVwS8NyPO+fQNxORoGnMnN/XhVJZvpqyKRUji9R0d8D66bYxUUsabXjP2y4HTVzbZtnvXFZZshk0cOtJjjekpYJaLK2esPR/iXwvSltNkrDQDPN/RmgEEMIevW8AgrPsqrnybFHxTpd7rEUHXBOe4nMNRIg3XHykB6jZk8q5bBPUe3I/f0DK5TJEBpTc6dO3P/j93u55VUqr39/SPRHnld2mCw+c8v6UOhsssO/DIZFPScD3DYqsk2N+/nz9zXfcOTdWGhawgxuIo1DTokrNQbG3pDrLqcWgqj13vqJFCmRA0O2CQIwJePd6+Np/XO3Uh/KL6FlQAAAQ8AAAAHc3NoLXJzYQAAAQC6sKEQHyl954BQn2BXuTgOB3NkENBxN7SD8ZaS8PNkDESytLjSIqrzoE6m7xuzprA+G23XRrCY/um3UvM7+7+zbwig2NIBbGbp3QFliQHegQKW6hTZP09jAQZk5jRrrEr/QT/s+gtHPmjxJK7XOQYxhInDKj+aJg62ExcwpQlP/0ATKNOIkdzTzzq916p0UOnnVaaPMKibh5Lv69GafIhKJRZSuuLN9fvs1G1RuUbxn/BNSeoRCr54L++Ztg09fJxunoyELs8mwgzCgB3pdZoUR2Z6ak05W4mvH3lkSz2BKUrlwxI6mterxhJy1GuN1K/zBG0gEMl2UTLajGK3qKM8 itbitloaner@MacBook-Pro-4.fios-router.home`
const CLIENT_PEM = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEAsruexO8yFkchFrmojoopDkmShY28C5KRPX26BXyy7DVCY+M8
PuW2VCbc1ivK6D44C/LNaD53K6OcLk7SNjbuH5KXhStB2996OhMpLLo+DpuPulHp
p9UBHG9G2aivGcoPQ4PXXj+y3hNL+/G+f42WnxYMMb/v7DITND0Q9+DQg2HGJpJB
5b0LUFhQHWz1woIITm3nWcmHaKmVboU7IjzFrUEv3HPTOLb+pjzrkJYWbd0fzof5
MFMruOaBkW+W4TuCYnKMxUt6HgpaVPlDG4LJZMwKSasWQVje/jwxUmYSpNhrJcns
j+LB9xN9GdNMlYv5UGIJkLAfh4OgpXpjUDDbvwIDAQABAoIBAEu2ctFVyk/pnbi0
uRR4rl+hBvKQUeJNGj2ELvL4Ggs5nIAX2IOEZ7JKLC6FqpSrFq7pEd5g57aSvixX
s3DH4CN7w7fj1ShBCNPlHgIWewdRGpeA74vrDWdwNAEsFdDE6aZeCTOhpDGy1vNJ
OrtpzS5i9pN0jTvvEneEjtWSZIHiiVlN+0hsFaiwZ6KXON+sDccZPmnP6Fzwj5Rc
WS0dKSwnxnx0otWgwWFs8nr306nSeMsNmQkHsS9lz4DEVpp9owdzrX1JmbQvNYAV
ohmB3ET4JYFgerqPXJfed9poueGuWCP6MYhsjNeHN35QhofxdO5/0i3JlZfqwZei
tNq/0oECgYEA6SqjRqDiIp3ajwyB7Wf0cIQG/P6JZDyN1jl//htgniliIH5UP1Tm
uAMG5MincV6X9lOyXyh6Yofu5+NR0yt9SqbDZVJ3ZCxKTun7pxJvQFd7wl5bMkiJ
qVfS08k6gQHHDoO+eel+DtpIfWc+e3tvX0aihSU0GZEMqDXYkkphLGECgYEAxDxb
+JwJ3N5UEjjkuvFBpuJnmjIaN9HvQkTv3inlx1gLE4iWBZXXsu4aWF8MCUeAAZyP
42hQDSkCYX/A22tYCEn/jfrU6A+6rkWBTjdUlYLvlSkhosSnO+117WEItb5cUE95
hF4UY7LNs1AsDkV4WE87f/EjpxSwUAjB2Lfd/B8CgYAJ/JiHsuZcozQ0Qk3iVDyF
ATKnbWOHFozgqw/PW27U92LLj32eRM2o/gAylmGNmoaZt1YBe2NaiwXxiqv7hnZU
VzYxRcn1UWxRWvY7Xq/DKrwTRCVVzwOObEOMbKcD1YaoGX50DEso6bKHJH/pnAzW
INlfKIvFuI+5OK0w/tyQoQKBgQCf/jpaOxaLfrV62eobRQJrByLDBGB97GsvU7di
IjTWz8DQH0d5rE7d8uWF8ZCFrEcAiV6DYZQK9smbJqbd/uoacAKtBro5rkFdPwwK
8m/DKqsdqRhkdgOHh7bjYH7Sdy8ax4Fi27WyB6FQtmgFBrz0+zyetsODwQlzZ4Bs
qpSRrwKBgQC0vWHrY5aGIdF+b8EpP0/SSLLALpMySHyWhDyxYcPqdhszYbjDcavv
xrrLXNUD2duBHKPVYE+7uVoDkpZXLUQ4x8argo/IwQM6Kh2ma1y83TYMT6XhL1+B
5UPcl6RXZBCkiU7nFIG6/0XKFqVWc3fU8e09X+iJwXIJ5Jatywtg+g==
-----END RSA PRIVATE KEY-----
`

func TestCertificateBasedAuth(t *testing.T) {
	signer, err := ssh.ParsePrivateKey([]byte(SERVER_PEM))
	if err != nil {
		t.Fatalf("unable to parse private key: %v", err)
	}
	address := newMockLineServer(t, signer, CLIENT_CERT_SIGNED_BY_SERVER)
	host, p, _ := net.SplitHostPort(address)
	port, _ := strconv.Atoi(p)

	connInfo := &connectionInfo{
		User:        "user",
		Host:        host,
		PrivateKey:  CLIENT_PEM,
		Certificate: CLIENT_CERT_SIGNED_BY_SERVER,
		Port:        port,
		Timeout:     "30s",
	}

	cfg, err := prepareSSHConfig(connInfo)
	if err != nil {
		t.Fatal(err)
	}

	c := &Communicator{
		connInfo: connInfo,
		config:   cfg,
	}

	var cmd remote.Cmd
	stdout := new(bytes.Buffer)
	cmd.Command = "echo foo"
	cmd.Stdout = stdout

	if err := c.Start(&cmd); err != nil {
		t.Fatal(err)
	}
	if err := c.Disconnect(); err != nil {
		t.Fatal(err)
	}
}

func TestAccUploadFile(t *testing.T) {
	// use the local ssh server and scp binary to check uploads
	if ok := os.Getenv("SSH_UPLOAD_TEST"); ok == "" {
		t.Log("Skipping Upload Acceptance without SSH_UPLOAD_TEST set")
		t.Skip()
	}

	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":    "ssh",
				"user":    os.Getenv("USER"),
				"host":    "127.0.0.1",
				"port":    "22",
				"timeout": "30s",
			},
		},
	}

	c, err := New(r)
	if err != nil {
		t.Fatalf("error creating communicator: %s", err)
	}

	tmpDir, err := ioutil.TempDir("", "communicator")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	content := []byte("this is the file content")
	source := bytes.NewReader(content)
	tmpFile := filepath.Join(tmpDir, "tempFile.out")
	err = c.Upload(tmpFile, source)
	if err != nil {
		t.Fatalf("error uploading file: %s", err)
	}

	data, err := ioutil.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(data, content) {
		t.Fatalf("bad: %s", data)
	}
}

func TestAccHugeUploadFile(t *testing.T) {
	// use the local ssh server and scp binary to check uploads
	if ok := os.Getenv("SSH_UPLOAD_TEST"); ok == "" {
		t.Log("Skipping Upload Acceptance without SSH_UPLOAD_TEST set")
		t.Skip()
	}

	r := &terraform.InstanceState{
		Ephemeral: terraform.EphemeralState{
			ConnInfo: map[string]string{
				"type":    "ssh",
				"user":    os.Getenv("USER"),
				"host":    "127.0.0.1",
				"port":    "22",
				"timeout": "30s",
			},
		},
	}

	c, err := New(r)
	if err != nil {
		t.Fatalf("error creating communicator: %s", err)
	}

	// copy 4GB of data, random to prevent compression.
	size := int64(1 << 32)
	source := io.LimitReader(rand.New(rand.NewSource(0)), size)

	dest, err := ioutil.TempFile("", "communicator")
	if err != nil {
		t.Fatal(err)
	}
	destName := dest.Name()
	dest.Close()
	defer os.Remove(destName)

	t.Log("Uploading to", destName)

	// bypass the Upload method so we can directly supply the file size
	// preventing the extra copy of the huge file.
	targetDir := filepath.Dir(destName)
	targetFile := filepath.Base(destName)

	scpFunc := func(w io.Writer, stdoutR *bufio.Reader) error {
		return scpUploadFile(targetFile, source, w, stdoutR, size)
	}

	err = c.scpSession("scp -vt "+targetDir, scpFunc)
	if err != nil {
		t.Fatal(err)
	}

	// check the final file size
	fs, err := os.Stat(destName)
	if err != nil {
		t.Fatal(err)
	}

	if fs.Size() != size {
		t.Fatalf("expected file size of %d, got %d", size, fs.Size())
	}
}

func TestScriptPath(t *testing.T) {
	cases := []struct {
		Input   string
		Pattern string
	}{
		{
			"/tmp/script.sh",
			`^/tmp/script\.sh$`,
		},
		{
			"/tmp/script_%RAND%.sh",
			`^/tmp/script_(\d+)\.sh$`,
		},
	}

	for _, tc := range cases {
		r := &terraform.InstanceState{
			Ephemeral: terraform.EphemeralState{
				ConnInfo: map[string]string{
					"type":        "ssh",
					"script_path": tc.Input,
				},
			},
		}
		comm, err := New(r)
		if err != nil {
			t.Fatalf("err: %s", err)
		}
		output := comm.ScriptPath()

		match, err := regexp.Match(tc.Pattern, []byte(output))
		if err != nil {
			t.Fatalf("bad: %s\n\nerr: %s", tc.Input, err)
		}
		if !match {
			t.Fatalf("bad: %s\n\n%s", tc.Input, output)
		}
	}
}

func TestScriptPath_randSeed(t *testing.T) {
	// Pre GH-4186 fix, this value was the deterministic start the pseudorandom
	// chain of unseeded math/rand values for Int31().
	staticSeedPath := "/tmp/terraform_1298498081.sh"
	c, err := New(&terraform.InstanceState{})
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	path := c.ScriptPath()
	if path == staticSeedPath {
		t.Fatalf("rand not seeded! got: %s", path)
	}
}

const testClientPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAxOgNXOJ/jrRDxBZTSk2X9otNy9zcpUmJr5ifDi5sy7j2ZiQS
beBt1Wf+tLNWis8Cyq06ttEvjjRuM75yucyD6GrqDTXVCSm4PeOIQeDhPhw26wYZ
O0h/mFgrAiCwaEl8AFXVBInOhVn/0nqgUpkwckh76bjTsNeifkiugK3cfJOuBdrU
ZGbgugJjmYo4Cmv7ECo1gCFT5N+BAjOji3z3N5ClBH5HaWC77jH7kTH0k5cZ+ZRQ
tG9EqLyvWlnzTAR/Yly0JInkOa16Ms1Au5sIJwEoJfHKsNVK06IlLob53nblwYW0
H5gv1Kb/rS+nUkpPtA5YFShB7iZnPLPPv6qXSwIDAQABAoIBAC0UY1rMkB9/rbQK
2G6+bPgI1HrDydAdkeQdsOxyPH43jlG8GGwHYZ3l/S4pkLqewijcmACay6Rm5IP8
Kg/XfquLLqJvnKJIZuHkYaGTdn3dv8T21Hf6FRwvs0j9auW1TSpWfDpZwmpNPIBX
irTeVXUUmynbIrvt4km/IhRbuYrbbb964CLYD1DCl3XssXxoRNvPpc5EtOuyDorA
5g1hvZR1FqbOAmOuNQMYJociMuWB8mCaHb+o1Sg4A65OLXxoKs0cuwInJ/n/R4Z3
+GrV+x5ypBMxXgjjQtKMLEOujkvxs1cp34hkbhKMHHXxbMu5jl74YtGGsLLk90rq
ieZGIgECgYEA49OM9mMCrDoFUTZdJaSARA/MOXkdQgrqVTv9kUHee7oeMZZ6lS0i
bPU7g+Bq+UAN0qcw9x992eAElKjBA71Q5UbZYWh29BDMZd8bRJmwz4P6aSMoYLWI
Sr31caJU9LdmPFatarNeehjSJtlTuoZD9+NElnnUwNaTeOOo5UdhTQsCgYEA3UGm
QWoDUttFwK9oL2KL8M54Bx6EzNhnyk03WrqBbR7PJcPKnsF0R/0soQ+y0FW0r8RJ
TqG6ze5fUJII72B4GlMTQdP+BIvaKQttwWQTNIjbbv4NksF445gdVOO1xi9SvQ7k
uvMVxOb+1jL3HAFa3furWu2tJRDs6dhuaILLxsECgYEAhnhlKUBDYZhVbxvhWsh/
lKymY/3ikQqUSX7BKa1xPiIalDY3YDllql4MpMgfG8L85asdMZ96ztB0o7H/Ss/B
IbLxt5bLLz+DBVXsaE82lyVU9h10RbCgI01/w3SHJHHjfBXFAcehKfvgfmGkE+IP
2A5ie1aphrCgFqh5FetNuQUCgYEAibL42I804FUtFR1VduAa/dRRqQSaW6528dWa
lLGsKRBalUNEEAeP6dmr89UEUVp1qEo94V0QGGe5FDi+rNPaC3AWdQqNdaDgNlkx
hoFU3oYqIuqj4ejc5rBd2N4a2+vJz3W8bokozDGC+iYf2mMRfUPKwj1XW9Er0OFs
3UhBsEECgYEAto/iJB7ZlCM7EyV9JW0tsEt83rbKMQ/Ex0ShbBIejej0Xx7bwx60
tVgay+bzJnNkXu6J4XVI98A/WsdI2kW4hL0STYdHV5HVA1l87V4ZbvTF2Bx8a8RJ
OF3UjpMTWKqOprw9nAu5VuwNRVzORF8ER8rgGeaR2/gsSvIYFy9VXq8=
-----END RSA PRIVATE KEY-----`

var testClientPublicKey = `ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDE6A1c4n+OtEPEFlNKTZf2i03L3NylSYmvmJ8OLmzLuPZmJBJt4G3VZ/60s1aKzwLKrTq20S+ONG4zvnK5zIPoauoNNdUJKbg944hB4OE+HDbrBhk7SH+YWCsCILBoSXwAVdUEic6FWf/SeqBSmTBySHvpuNOw16J+SK6Ardx8k64F2tRkZuC6AmOZijgKa/sQKjWAIVPk34ECM6OLfPc3kKUEfkdpYLvuMfuRMfSTlxn5lFC0b0SovK9aWfNMBH9iXLQkieQ5rXoyzUC7mwgnASgl8cqw1UrToiUuhvneduXBhbQfmC/Upv+tL6dSSk+0DlgVKEHuJmc8s8+/qpdL`

func acceptUserPass(goodUser, goodPass string) func(ssh.ConnMetadata, []byte) (*ssh.Permissions, error) {
	return func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
		if c.User() == goodUser && string(pass) == goodPass {
			return nil, nil
		}
		return nil, fmt.Errorf("password rejected for %q", c.User())
	}
}

func acceptPublicKey(keystr string) func(ssh.ConnMetadata, ssh.PublicKey) (*ssh.Permissions, error) {
	return func(_ ssh.ConnMetadata, inkey ssh.PublicKey) (*ssh.Permissions, error) {
		goodkey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keystr))
		if err != nil {
			return nil, fmt.Errorf("error parsing key: %v", err)
		}

		if bytes.Equal(inkey.Marshal(), goodkey.Marshal()) {
			return nil, nil
		}

		return nil, fmt.Errorf("public key rejected")
	}
}
