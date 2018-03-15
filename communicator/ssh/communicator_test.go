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

func newMockLineServer(t *testing.T, signer ssh.Signer) string {
	serverConfig := &ssh.ServerConfig{
		PasswordCallback:  acceptUserPass("user", "pass"),
		PublicKeyCallback: acceptPublicKey(testClientPublicKey),
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
	address := newMockLineServer(t, nil)
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
	address := newMockLineServer(t, nil)
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

func TestLostConnection(t *testing.T) {
	address := newMockLineServer(t, nil)
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

	cmd.Wait()
	if cmd.Err() == nil {
		t.Fatal("expected communicator error")
	}
	if cmd.ExitStatus() != 0 {
		t.Fatal("command should not have returned an exit status")
	}
}

func TestHostKey(t *testing.T) {
	// get the server's public key
	signer, err := ssh.ParsePrivateKey([]byte(testServerPrivateKey))
	if err != nil {
		panic("unable to parse private key: " + err.Error())
	}
	pubKey := fmt.Sprintf("ssh-rsa %s", base64.StdEncoding.EncodeToString(signer.PublicKey().Marshal()))

	address := newMockLineServer(t, nil)
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
	address = newMockLineServer(t, nil)
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

	address := newMockLineServer(t, signer)
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
	address = newMockLineServer(t, signer)
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
	goodkey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keystr))
	if err != nil {
		panic(fmt.Errorf("error parsing key: %s", err))
	}
	return func(_ ssh.ConnMetadata, inkey ssh.PublicKey) (*ssh.Permissions, error) {
		if bytes.Equal(inkey.Marshal(), goodkey.Marshal()) {
			return nil, nil
		}

		return nil, fmt.Errorf("public key rejected")
	}
}
