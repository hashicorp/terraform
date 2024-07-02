// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package terraform

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"unsafe"

	"github.com/zclconf/go-cty/cty"
	"golang.org/x/crypto/ssh"

	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/providers"
	"github.com/hashicorp/terraform/internal/tfdiags"
)

func ephemeralSSHTunnelsSchema() providers.Schema {
	return providers.Schema{
		Block: &configschema.Block{
			Attributes: map[string]*configschema.Attribute{
				"server":   {Type: cty.String, Required: true},
				"username": {Type: cty.String, Required: true},

				"auth_methods": {
					Type: cty.List(
						// This object type is acting like a sum type rather
						// than a product type, requiring that exactly one
						// of its attributes is set to decide which member
						// to instantiate.
						cty.Object(map[string]cty.Type{
							"password": cty.String,
							// TODO: SSH keys, etc
						}),
					),
					Required: true,
				},

				"tcp_to_remote": {
					Type: cty.Map(cty.Object(map[string]cty.Type{
						"local_host": cty.String,
						"local_port": cty.String,
						"local":      cty.String,
						"remote":     cty.String,
					})),
					Computed: true,
				},
				"tcp_from_remote": {
					Type: cty.Map(cty.Object(map[string]cty.Type{
						"remote_port": cty.String,
					})),
					Computed: true,
				},
			},
			BlockTypes: map[string]*configschema.NestedBlock{
				"tcp_local_to_remote": {
					Nesting: configschema.NestingMap,
					Block: configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"remote": {Type: cty.String, Required: true},
							"local":  {Type: cty.String, Optional: true},
						},
					},
				},
				"tcp_remote_to_local": {
					Nesting: configschema.NestingMap,
					Block: configschema.Block{
						Attributes: map[string]*configschema.Attribute{
							"remote": {Type: cty.String, Required: true},
							"local":  {Type: cty.String, Required: true},
						},
					},
				},
			},
		},
	}
}

type ephemeralSSHTunnelsConns struct {
	// Keys here are the addresses of the corresponding ephemeralSSHTunnelState
	// objects. This probably isn't a good idea in the long run, but it's
	// fine for a prototype.
	active map[uintptr]*ephemeralSSHTunnelConn
	mu     sync.Mutex
}

type ephemeralSSHTunnelConn struct {
	client         *ssh.Client
	closeListeners func()
}

var ephemeralSSHTunnels ephemeralSSHTunnelsConns

func init() {
	ephemeralSSHTunnels.mu.Lock()
	ephemeralSSHTunnels.active = make(map[uintptr]*ephemeralSSHTunnelConn)
	ephemeralSSHTunnels.mu.Unlock()
}

func openEphemeralSSHTunnels(req providers.OpenEphemeralRequest) providers.OpenEphemeralResponse {
	log.Printf("[TRACE] terraform_ssh_tunnels: opening connection")
	var resp providers.OpenEphemeralResponse

	serverAddr, clientConfig, diags := makeEphemeralSSHTunnelClientConfig(req.Config)
	resp.Diagnostics = resp.Diagnostics.Append(diags)
	if diags.HasErrors() {
		return resp
	}
	log.Printf("[DEBUG] terraform_ssh_tunnels: connecting to %s as %q", serverAddr, clientConfig.User)
	client, err := ssh.Dial("tcp", serverAddr, clientConfig)
	if err != nil {
		resp.Diagnostics = resp.Diagnostics.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Can't connect to SSH server",
			fmt.Sprintf("Failed to connect to SSH server to establish tunnels: %s.", err),
			nil, // A number of different arguments could potentially cause a connection failure
		))
		return resp
	}

	ephemeralSSHTunnels.mu.Lock()
	defer ephemeralSSHTunnels.mu.Unlock()

	conn := &ephemeralSSHTunnelConn{
		client: client,
	}
	connID := uintptr(unsafe.Pointer(conn))
	ephemeralSSHTunnels.active[connID] = conn

	var intCtx bytes.Buffer
	intCtx.Grow(8)
	binary.Write(&intCtx, binary.LittleEndian, uint64(connID))
	resp.InternalContext = intCtx.Bytes()

	tcpToRemoteVals := map[string]cty.Value{}
	tcpFromRemoteVals := map[string]cty.Value{}

	ctx, cancel := context.WithCancel(context.Background())
	conn.closeListeners = cancel
	for it := req.Config.GetAttr("tcp_local_to_remote").ElementIterator(); it.Next(); {
		keyVal, configVal := it.Element()
		key := keyVal.AsString()
		log.Printf("[TRACE] terraform_ssh_tunnels: tcp_local_to_remote %q", key)

		// FIXME: The following is not robust against unknown values and
		// other such oddities.
		remoteAddr := configVal.GetAttr("remote").AsString()

		listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:0")
		if err != nil {
			resp.Diagnostics = resp.Diagnostics.Append(tfdiags.AttributeValue(
				tfdiags.Error,
				"Can't open TCP listen port",
				fmt.Sprintf("Failed to open local TCP listen port for tcp_local_to_remote %q: %s.", key, err),
				cty.GetAttrPath("tcp_local_to_remote").IndexString(key),
			))
			continue
		}

		go sshTunnelLocalToRemote(ctx, listener, client, remoteAddr)

		tcpToRemoteVals[key] = cty.ObjectVal(map[string]cty.Value{
			"local_host": cty.NullVal(cty.String), // TODO: Populate
			"local_port": cty.NullVal(cty.String), // TODO: Populate
			"local":      cty.StringVal(listener.Addr().String()),
			"remote":     cty.NullVal(cty.String), // TODO: Populate
		})
	}
	for it := req.Config.GetAttr("tcp_remote_to_local").ElementIterator(); it.Next(); {
		log.Printf("[TRACE] terraform_ssh_tunnels: tcp_remote_to_local")

		// TODO: Implement
	}

	var tcpToRemoteVal cty.Value
	if len(tcpToRemoteVals) != 0 {
		tcpToRemoteVal = cty.MapVal(tcpToRemoteVals)
	} else {
		tcpToRemoteVal = cty.MapValEmpty(cty.Object(map[string]cty.Type{
			"local_host": cty.String,
			"local_port": cty.String,
			"local":      cty.String,
			"remote":     cty.String,
		}))
	}
	var tcpFromRemoteVal cty.Value
	if len(tcpFromRemoteVals) != 0 {
		tcpFromRemoteVal = cty.MapVal(tcpFromRemoteVals)
	} else {
		tcpFromRemoteVal = cty.MapValEmpty(cty.Object(map[string]cty.Type{
			"remote_port": cty.String,
		}))
	}

	resp.Result = cty.ObjectVal(map[string]cty.Value{
		"server":              req.Config.GetAttr("server"),
		"username":            req.Config.GetAttr("username"),
		"auth_methods":        req.Config.GetAttr("auth_methods"),
		"tcp_local_to_remote": req.Config.GetAttr("tcp_local_to_remote"),
		"tcp_remote_to_local": req.Config.GetAttr("tcp_remote_to_local"),

		"tcp_to_remote":   tcpToRemoteVal,
		"tcp_from_remote": tcpFromRemoteVal,
	})

	return resp
}

func renewEphemeralSSHTunnels(req providers.RenewEphemeralRequest) providers.RenewEphemeralResponse {
	// SSH tunnel connections don't need to be explicitly renewed, so this
	// should never get called. (The SSH library handles keepalives internally
	// itself, without our help.)
	return providers.RenewEphemeralResponse{}
}

func closeEphemeralSSHTunnels(req providers.CloseEphemeralRequest) providers.CloseEphemeralResponse {
	log.Printf("[TRACE] terraform_ssh_tunnels: closing connection")
	var resp providers.CloseEphemeralResponse

	intCtx := bytes.NewReader(req.InternalContext)
	var connIDInt uint64
	if err := binary.Read(intCtx, binary.LittleEndian, &connIDInt); err != nil {
		// Should not get here if the client is behaving correctly, because
		// we should only get InternalContext values that we returned previously
		// from [openEphemeralSSHTunnels].
		resp.Diagnostics = resp.Diagnostics.Append(err)
		return resp
	}
	connID := uintptr(connIDInt)

	ephemeralSSHTunnels.mu.Lock()
	defer ephemeralSSHTunnels.mu.Unlock()

	conn, ok := ephemeralSSHTunnels.active[connID]
	if !ok {
		// Should not get here because client should only pass InternalContext
		// values that we returned previously from [openEphemeralSSHTunnels].
		resp.Diagnostics = resp.Diagnostics.Append(fmt.Errorf("trying to close unknown connection %#v", connID))
		return resp
	}

	if conn.closeListeners != nil {
		conn.closeListeners()
	}

	err := conn.client.Close()
	if err != nil {
		// Perhaps the connection already got terminated exceptionally before
		// we got around to closing it?
		resp.Diagnostics = resp.Diagnostics.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"Could not close SSH connection",
			fmt.Sprintf("Failed to close tunnel SSH connection: %s.", err),
			nil,
		))
	}
	// We'll delete it even if we failed to close it, because we're not going
	// to get any opportunity to do anything with it again anyway, and it
	// seems to be somehow broken.
	delete(ephemeralSSHTunnels.active, connID)

	return resp
}

func makeEphemeralSSHTunnelClientConfig(configVal cty.Value) (serverAddr string, clientConfig *ssh.ClientConfig, diags tfdiags.Diagnostics) {
	clientConfig = &ssh.ClientConfig{}

	// FIXME: In a real implementation we ought to constrain this better,
	// such as by having the configuration include a set of allowed host
	// keys.
	clientConfig.HostKeyCallback = ssh.InsecureIgnoreHostKey()

	if serverVal := configVal.GetAttr("server"); serverVal.IsKnown() {
		serverAddr = serverVal.AsString()
	} else {
		// FIXME: Terrible error message just for prototype.
		// In a real implementation we would hopefully be able to "defer"
		// this, but deferred actions is being implemented concurrently with
		// this prototype and so this is best to avoid conflicting with that
		// other project.
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"SSH server address not known",
			"The SSH server address is derived from a value that isn't known yet.",
			cty.GetAttrPath("server"),
		))
	}
	if usernameVal := configVal.GetAttr("username"); usernameVal.IsKnown() {
		clientConfig.User = usernameVal.AsString()
	} else {
		// FIXME: Terrible error message just for prototype.
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"SSH username not known",
			"The username is derived from a value that isn't known yet.",
			cty.GetAttrPath("server"),
		))
	}

	if authMethodsVal := configVal.GetAttr("auth_methods"); authMethodsVal.IsWhollyKnown() {
		for it := authMethodsVal.ElementIterator(); it.Next(); {
			idx, authMethodObj := it.Element()
			if authMethodObj.IsNull() {
				continue // FIXME: should probably be an error, actually
			}

			// The following makes sure that exactly one attribute is set
			// and checks which it is. This pattern treats the object type
			// as a sum type rather than as a product type.
			var attrName string
			var attrVal cty.Value
			for n := range authMethodObj.Type().AttributeTypes() {
				val := authMethodObj.GetAttr(n)
				if val.IsNull() {
					continue
				}
				if attrName != "" {
					diags = diags.Append(tfdiags.AttributeValue(
						tfdiags.Error,
						"Ambiguous auth method selection",
						fmt.Sprintf("Cannot set both %q and %q.", attrName, n),
						cty.GetAttrPath("auth_methods").Index(idx),
					))
					continue
				}
				attrName = n
				attrVal = val
			}
			if attrName == "" {
				diags = diags.Append(tfdiags.AttributeValue(
					tfdiags.Error,
					"No auth method selection",
					"Must set one of the possible attributes to select the auth method type.",
					cty.GetAttrPath("auth_methods").Index(idx),
				))
				continue
			}

			switch attrName {
			case "password":
				if attrVal.IsNull() {
					diags = diags.Append(tfdiags.AttributeValue(
						tfdiags.Error,
						"Password cannot be null",
						"When authenticating using a password, the password must be specified.",
						cty.GetAttrPath("auth_methods").Index(idx).GetAttr("password"),
					))
					continue
				}
				clientConfig.Auth = append(clientConfig.Auth, ssh.Password(attrVal.AsString()))
			}
		}
	} else {
		// FIXME: Terrible error message just for prototype.
		diags = diags.Append(tfdiags.AttributeValue(
			tfdiags.Error,
			"SSH server auth methods not known",
			"The auth_methods structure contains unknown values.",
			cty.GetAttrPath("auth_methods"),
		))
	}

	return serverAddr, clientConfig, diags
}

func sshTunnelLocalToRemote(ctx context.Context, listener net.Listener, sshClient *ssh.Client, remoteAddr string) {
	log.Printf("[TRACE] terraform_ssh_tunnels: forwarding connections from %s to %s", listener.Addr(), remoteAddr)

	for {
		localConn, err := listener.Accept()
		if err != nil {
			log.Printf("[DEBUG] terraform_ssh_tunnels: error accepting connection from %s: %s", listener.Addr(), err)
			break
		}

		remoteConn, err := sshClient.DialContext(ctx, "tcp", remoteAddr)
		if err != nil {
			localConn.Close()
			log.Printf("[DEBUG] terraform_ssh_tunnels: error opening connection to %s: %s", remoteAddr, err)
			break
		}

		localConnTCP := localConn.(*net.TCPConn)

		// If we managed to open both connections then we just need to pass
		// arbitrary bytes between them for as long as they're both open.
		go func() {
			var wg sync.WaitGroup

			log.Printf("[TRACE] terraform_ssh_tunnels: tunnel connection %s->%s: open", localConnTCP.LocalAddr(), remoteConn.RemoteAddr())

			wg.Add(2)
			go func() {
				io.Copy(localConnTCP, remoteConn)
				localConnTCP.CloseWrite()
				wg.Done()
			}()
			go func() {
				io.Copy(remoteConn, localConnTCP)
				// SSH tunnel client conn doesn't support CloseWrite, so
				// we can't signal that nothing more is coming on that one.
				wg.Done()
			}()

			wg.Wait()
			localConnTCP.Close()
			remoteConn.Close()

			log.Printf("[TRACE] terraform_ssh_tunnels: tunnel connection %s->%s: closed", localConnTCP.LocalAddr(), remoteConn.RemoteAddr())
		}()
	}
}
