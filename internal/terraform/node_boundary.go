package terraform

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net"
	"runtime"
	"sync"

	"github.com/hashicorp/boundary/api"
	"github.com/hashicorp/boundary/api/authtokens"
	"github.com/hashicorp/boundary/api/targets"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/terraform/internal/addrs"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/dag"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/vault/sdk/helper/base62"
	nkeyring "github.com/jefferai/keyring"
	zkeyring "github.com/zalando/go-keyring"
	"github.com/zclconf/go-cty/cty"
	"go.uber.org/atomic"
	"nhooyr.io/websocket"
)

const (
	NoneKeyring          = "none"
	AutoKeyring          = "auto"
	WincredKeyring       = "wincred"
	PassKeyring          = "pass"
	KeychainKeyring      = "keychain"
	SecretServiceKeyring = "secret-service"

	DefaultTokenName = "default"
	LoginCollection  = "login"
	PassPrefix       = "HashiCorp_Boundary"
	StoredTokenName  = "HashiCorp Boundary Auth Token"
)

type config struct {
	targetId           string
	targetName         string
	targetScopeId      string
	targetScopeName    string
	authorizationToken string
	hostId             string
	listenAddr         net.IP
	listenPort         int
}

type NodeBoundary struct {
	ConnectionName string
	Config         hcl.Body
	Connection     hcl.Body
	Schema         *configschema.Block
	DeclRange      *hcl.Range
	CloseNode      *NodeBoundaryCloser
}

func (n *NodeBoundary) Name() string {
	return "boundary." + n.ConnectionName
}

// getClient creates the client based on the configuration given by the user
// and fetches the token from the keyring if needed
func (n *NodeBoundary) getClient(ctx EvalContext) (client *api.Client, diags tfdiags.Diagnostics) {
	config, err := api.DefaultConfig()
	if err != nil {
		return nil, diags.Append(err)
	}

	val, _, moreDiags := ctx.EvaluateBlock(n.Config, n.Schema, nil, EvalDataForNoInstanceKey)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return nil, diags
	}

	m := val.AsValueMap()
	delete(m, "connection")
	configVal := cty.ObjectVal(m)

	if !configVal.IsWhollyKnown() {
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid Boundary connection configuration",
			Detail:   "The configuration depends on values that cannot be determined until apply.",
			Subject:  n.DeclRange,
		})
	}

	if addr := configVal.GetAttr("address"); !addr.IsNull() {
		config.Addr = addr.AsString()
	}
	if caCert := configVal.GetAttr("ca_cert"); !caCert.IsNull() {
		config.TLSConfig.CACert = caCert.AsString()
	}
	if caPath := configVal.GetAttr("ca_path"); !caPath.IsNull() {
		config.TLSConfig.CAPath = caPath.AsString()
	}
	if clientCert := configVal.GetAttr("client_cert"); !clientCert.IsNull() {
		config.TLSConfig.ClientCert = clientCert.AsString()
	}
	if clientKey := configVal.GetAttr("client_key"); !clientKey.IsNull() {
		config.TLSConfig.ClientKey = clientKey.AsString()
	}
	if tlsInsecure := configVal.GetAttr("tls_insecure"); !tlsInsecure.IsNull() {
		config.TLSConfig.Insecure = tlsInsecure.True()
	}
	if tlsServerName := configVal.GetAttr("tls_server_name"); !tlsServerName.IsNull() {
		config.TLSConfig.ServerName = tlsServerName.AsString()
	}
	if err := config.ConfigureTLS(); err != nil {
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to set Boundary client TLS configuration",
			Detail:   err.Error(),
			Subject:  n.DeclRange,
		})
	}

	var token, keyringType string
	if attr := configVal.GetAttr("token"); !attr.IsNull() {
		token = val.AsString()
	}
	if attr := val.GetAttr("keyring_type"); !attr.IsNull() {
		keyringType = val.AsString()
	} else {
		keyringType = "auto"
	}

	switch {
	case token != "":
		config.Token = token
	case keyringType != "" && keyringType != "none":
		var tokenName string
		if attr := val.GetAttr("token_name"); attr.IsNull() {
			tokenName = DefaultTokenName
		} else {
			tokenName = attr.AsString()
		}

		var foundKeyringType bool
		switch runtime.GOOS {
		case "windows":
			switch keyringType {
			case AutoKeyring, WincredKeyring, PassKeyring:
				foundKeyringType = true
				if keyringType == AutoKeyring {
					keyringType = WincredKeyring
				}
			}
		case "darwin":
			switch keyringType {
			case AutoKeyring, KeychainKeyring, PassKeyring:
				foundKeyringType = true
				if keyringType == AutoKeyring {
					keyringType = KeychainKeyring
				}
			}
		default:
			switch keyringType {
			case AutoKeyring, SecretServiceKeyring, PassKeyring:
				foundKeyringType = true
				if keyringType == AutoKeyring {
					keyringType = PassKeyring
				}
			}
		}

		if !foundKeyringType {
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Given keyring type %q is not valid, or not valid for this platform", keyringType),
				Subject:  n.DeclRange,
			})
		}

		var available bool
		switch keyringType {
		case WincredKeyring, KeychainKeyring:
			available = true
		case PassKeyring, SecretServiceKeyring:
			avail := nkeyring.AvailableBackends()
			for _, a := range avail {
				if keyringType == string(a) {
					available = true
				}
			}
		}

		if !available {
			return nil, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Keyring type %q is not available on this machine. For help with setting up keyrings, see https://www.boundaryproject.io/docs/api-clients/cli.", keyringType),
				Subject:  n.DeclRange,
			})
		}

		switch keyringType {
		case WincredKeyring, KeychainKeyring:
			token, err = zkeyring.Get(StoredTokenName, tokenName)
			if err != nil {
				if err == zkeyring.ErrNotFound {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  "No saved credential found, continuing without",
						Subject:  n.DeclRange,
					})
				} else {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  fmt.Sprintf("Error reading auth token from keyring: %s", err),
						Detail:   "Token must be provided via BOUNDARY_TOKEN env var or token attribute. Reading the token can also be disabled via keyring_type=none.",
						Subject:  n.DeclRange,
					})
				}
				token = ""
			}

		default:
			krConfig := nkeyring.Config{
				LibSecretCollectionName: LoginCollection,
				PassPrefix:              PassPrefix,
				AllowedBackends:         []nkeyring.BackendType{nkeyring.BackendType(keyringType)},
			}

			kr, err := nkeyring.Open(krConfig)
			if err != nil {
				return nil, diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Error opening keyring: %s", err),
					Detail:   "Token must be provided via BOUNDARY_TOKEN env var or token attribute. Reading the token can also be disabled via keyring_type=none.",
					Subject:  n.DeclRange,
				})
			}

			item, err := kr.Get(tokenName)
			if err != nil {
				return nil, diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Error fetching token from keyring: %s", err),
					Detail:   "Token must be provided via BOUNDARY_TOKEN env var or token attribute. Reading the token can also be disabled via keyring_type=none.",
					Subject:  n.DeclRange,
				})
			}

			token = string(item.Data)
		}

		if token != "" {
			tokenBytes, err := base64.RawStdEncoding.DecodeString(token)
			switch {
			case err != nil:
				return nil, diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  fmt.Sprintf("Error base64-unmarshaling stored token from system credential store: %s", err),
					Subject:  n.DeclRange,
				})
			case len(tokenBytes) == 0:
				return nil, diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Zero length token after decoding stored token from system credential store",
					Subject:  n.DeclRange,
				})
			default:
				var authToken authtokens.AuthToken
				if err := json.Unmarshal(tokenBytes, &authToken); err != nil {
					return nil, diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  fmt.Sprintf("Error unmarshaling stored token information after reading from system credential store: %s", err),
						Subject:  n.DeclRange,
					})
				} else {
					config.Token = authToken.Token
				}
			}
		}
	}

	client, err = api.NewClient(config)
	if err != nil {
		return nil, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Failed to create Boundary client",
			Detail:   err.Error(),
			Subject:  n.DeclRange,
		})
	}
	return client, diags
}

// GraphNodeExecutable
// NodeBoundary.Execute is an Execute implementation that evaluates the
// configuration for a boundary connection, starts the proxy and writes the
// configuration to a transient part of the state.
func (n *NodeBoundary) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	switch op {
	case walkValidate:
		_, diags := n.validate(ctx)
		return diags
	case walkPlan, walkApply, walkPlanDestroy, walkDestroy, walkImport, walkEval:
		return n.StartProxy(ctx)
	default:
		return diags.Append(fmt.Errorf("unexpected walkOperation %s", op))
	}
}

func (n *NodeBoundary) validate(ctx EvalContext) (c config, diags tfdiags.Diagnostics) {
	val, _, moreDiags := ctx.EvaluateBlock(n.Connection, &n.Schema.BlockTypes["connection"].Block, nil, EvalDataForNoInstanceKey)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return
	}

	if !val.IsWhollyKnown() {
		return c, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Invalid Boundary connection configuration",
			Detail:   fmt.Sprintf("The configuration for %s depends on values that cannot be determined until apply.", n.ConnectionName),
			Subject:  n.DeclRange,
		})
	}

	if attr := val.GetAttr("authorization_token"); !attr.IsNull() {
		c.authorizationToken = attr.AsString()
	}
	if attr := val.GetAttr("target_id"); !attr.IsNull() {
		c.targetId = attr.AsString()
	}
	if attr := val.GetAttr("target_name"); !attr.IsNull() {
		c.targetName = attr.AsString()
	}
	if attr := val.GetAttr("target_scope_id"); !attr.IsNull() {
		c.targetScopeId = attr.AsString()
	}
	if attr := val.GetAttr("target_scope_name"); !attr.IsNull() {
		c.targetScopeName = attr.AsString()
	}
	if attr := val.GetAttr("host_id"); !attr.IsNull() {
		c.hostId = attr.AsString()
	}
	c.listenAddr = net.ParseIP("127.0.0.1")
	if addr := val.GetAttr("listen_addr"); !addr.IsNull() {
		c.listenAddr = net.ParseIP(addr.AsString())
		if c.listenAddr == nil {
			return c, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  fmt.Sprintf("Could not successfully parse listen_addr of %s", addr.AsString()),
				Subject:  n.DeclRange,
			})
		}
	}
	if port := val.GetAttr("listen_port"); port.IsNull() {
		return c, diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Incorrect Boundary connection configuration",
			Detail:   "listen_port must be specified",
			Subject:  n.DeclRange,
		})
	} else {
		i, _ := port.AsBigFloat().Int64()
		c.listenPort = int(i)
	}

	switch {
	case c.authorizationToken != "":
		switch {
		case c.targetId != "":
			return c, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "target_id and authorization_token cannot both be specified",
			})
		case c.targetName != "":
			return c, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "target_name and authorization_token cannot both be specified",
			})
		}
	default:
		if c.targetId == "" &&
			(c.targetName == "" ||
				(c.targetScopeId == "" && c.targetScopeName == "")) {
			return c, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Target ID was not passed in, but no combination of target name and scope ID/name was passed in either",
			})
		}
		if c.targetId != "" &&
			(c.targetName != "" || c.targetScopeId != "" || c.targetScopeName != "") {
			return c, diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Cannot specify a target ID and also other lookup parameters",
			})
		}
	}

	state := ctx.State()
	if state == nil {
		return c, diags.Append(fmt.Errorf("cannot write connection configuration to a nil state"))
	}

	state.SetBoundaryConnection(n.ConnectionName, cty.DynamicVal)

	return
}

func (n *NodeBoundary) StartProxy(ctx EvalContext) (diags tfdiags.Diagnostics) {
	client, diags := n.getClient(ctx)
	if diags != nil {
		return diags
	}

	c, moreDiags := n.validate(ctx)
	diags = diags.Append(moreDiags)
	if diags.HasErrors() {
		return diags
	}

	var credentials []cty.Value
	switch {
	case c.authorizationToken != "":
		if c.authorizationToken[0] == '{' {
			// Attempt to decode the JSON output of an authorize-session call
			// and pull the token out of there
			sessionAuthz := new(targets.SessionAuthorization)
			if err := json.Unmarshal([]byte(c.authorizationToken), sessionAuthz); err == nil {
				c.authorizationToken = sessionAuthz.AuthorizationToken
			}
		}
	default:
		tClient := targets.NewClient(client)

		var opts []targets.Option
		if c.hostId != "" {
			opts = append(opts, targets.WithHostId(c.hostId))
		}
		if c.targetName != "" {
			opts = append(opts, targets.WithName(c.targetName))
		}
		if c.targetScopeId != "" {
			opts = append(opts, targets.WithScopeId(c.targetId))
		}
		if c.targetScopeName != "" {
			opts = append(opts, targets.WithScopeName(c.targetScopeName))
		}

		sar, err := tClient.AuthorizeSession(context.Background(), c.targetId, opts...)
		if err != nil {
			if apiErr := api.AsServerError(err); apiErr != nil {
				return diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Error from controller when performing authorize-session action against given target",
					Detail:   apiErr.Message,
					Subject:  n.DeclRange,
				})
			}
			return diags.Append(&hcl.Diagnostic{
				Severity: hcl.DiagError,
				Summary:  "Error trying to authorize a session against target",
				Detail:   err.Error(),
				Subject:  n.DeclRange,
			})
		}
		c.authorizationToken = sar.Item.AuthorizationToken

		for _, c := range sar.Item.Credentials {
			s := map[string]cty.Value{}
			for k, v := range c.Secret.Decoded {
				switch v := v.(type) {
				case bool:
					s[k] = cty.BoolVal(v)
				case float64:
					s[k] = cty.NumberFloatVal(v)
				case string:
					s[k] = cty.StringVal(v)
				default:
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagWarning,
						Summary:  "Unexpected secret credential type",
						Detail:   fmt.Sprintf("%T is not supported", v),
					})
				}
			}
			credentials = append(credentials, cty.ObjectVal(s))
		}
	}

	sessionAuthzData, tlsConf, err := targets.AuthorizationToken(c.authorizationToken).GetConfig()
	if err != nil {
		return diags.Append(err)
	}
	connectionsLeft := atomic.NewInt32(sessionAuthzData.ConnectionLimit)

	transport := cleanhttp.DefaultTransport()
	transport.DisableKeepAlives = false
	transport.TLSClientConfig = tlsConf
	// This isn't/shouldn't used anyways really because the connection is
	// hijacked, just setting for completeness
	transport.IdleConnTimeout = 0

	listener, err := net.ListenTCP("tcp", &net.TCPAddr{
		IP:   c.listenAddr,
		Port: c.listenPort,
	})
	if err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Error starting listening port",
			Detail:   err.Error(),
			Subject:  n.DeclRange,
		})
	}

	n.CloseNode.RegisterCloser(listener.Close, connectionsLeft)

	tofuToken, err := base62.Random(20)
	if err != nil {
		return diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Could not derive random bytes for tofu token",
			Detail:   err.Error(),
			Subject:  n.DeclRange,
		})
	}

	connWg := new(sync.WaitGroup)
	connWg.Add(1)
	go func() {
		defer connWg.Done()
		for {
			listeningConn, err := listener.AcceptTCP()
			if err != nil {
				if connectionsLeft.Load() == 0 {
					return
				}
				n.CloseNode.ReportErr(err)
				continue
			}
			connWg.Add(1)
			go func() {
				defer listeningConn.Close()
				defer connWg.Done()
				wsConn, err := targets.AuthorizationToken(c.authorizationToken).Connect(context.Background(), transport)
				if err != nil {
					n.CloseNode.ReportErr(err)
				} else {

					if err := targets.Handshake(context.Background(), wsConn, tofuToken); err != nil {
						n.CloseNode.ReportErr(err)
					} else {
						// Get a wrapped net.Conn so we can use io.Copy
						netConn := websocket.NetConn(context.Background(), wsConn, websocket.MessageBinary)

						localWg := new(sync.WaitGroup)
						localWg.Add(2)

						go func() {
							defer localWg.Done()
							io.Copy(netConn, listeningConn)
							netConn.Close()
							listeningConn.Close()
						}()
						go func() {
							defer localWg.Done()
							io.Copy(listeningConn, netConn)
							listeningConn.Close()
							netConn.Close()
						}()
						localWg.Wait()
					}
				}
			}()
		}
	}()

	state := ctx.State()
	if state == nil {
		return diags.Append(fmt.Errorf("cannot write connection configuration to a nil state"))
	}

	state.SetBoundaryConnection(n.ConnectionName, cty.ObjectVal(map[string]cty.Value{
		"listen_addr":         cty.StringVal(c.listenAddr.String()),
		"listen_port":         cty.NumberVal(big.NewFloat(float64(c.listenPort))),
		"authorization_token": cty.StringVal(c.authorizationToken),
		"host_id":             cty.StringVal(c.hostId),
		"target_id":           cty.StringVal(c.targetId),
		"target_name":         cty.StringVal(c.targetName),
		"target_scope_id":     cty.StringVal(c.targetScopeId),
		"target_scope_name":   cty.StringVal(c.targetScopeName),
		"credentials":         cty.TupleVal(credentials),
	}))

	return
}

// GraphNodeModuleInstance
func (n *NodeBoundary) Path() addrs.ModuleInstance {
	return addrs.RootModuleInstance
}

// GraphNodeModulePath
func (n *NodeBoundary) ModulePath() addrs.Module {
	return addrs.RootModule
}

func (n *NodeBoundary) ReferenceableAddrs() []addrs.Referenceable {
	return []addrs.Referenceable{addrs.Boundary{Name: n.ConnectionName}}
}

// GraphNodeReferencer
func (n *NodeBoundary) References() []*addrs.Reference {
	refs := ReferencesFromConfig(n.Config, &configschema.Block{
		Attributes: n.Schema.Attributes,
	})

	refs = append(refs, ReferencesFromConfig(n.Connection, &n.Schema.BlockTypes["connection"].Block)...)

	return refs
}

// GraphNodeDotter impl.
func (n *NodeBoundary) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: name,
		Attrs: map[string]string{
			"label":     n.Name(),
		},
	}
}

type NodeBoundaryCloser struct {
	lock    *sync.Mutex
	closers []func() error
	errors  []error
	connectionLeft []*atomic.Int32
}

func (n *NodeBoundaryCloser) ReportErr(err error) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.errors = append(n.errors, err)
}

func (n *NodeBoundaryCloser) RegisterCloser(closer func() error, connectionsLeft *atomic.Int32) {
	n.lock.Lock()
	defer n.lock.Unlock()
	n.closers = append(n.closers, closer)
}

func (n *NodeBoundaryCloser) Execute(ctx EvalContext, op walkOperation) (diags tfdiags.Diagnostics) {
	n.lock.Lock()
	defer n.lock.Unlock()

	for _, c := range n.connectionLeft {
		c.Store(0)
	}

	for _, f := range n.closers {
		if err := f(); err != nil {
			diags = diags.Append(err)
		}
	}
	for _, err := range n.errors {
		diags = diags.Append(err)
	}
	return diags
}

func (n *NodeBoundaryCloser) Name() string {
	return "boundary closer"
}

func (n *NodeBoundaryCloser) DotNode(name string, opts *dag.DotOpts) *dag.DotNode {
	return &dag.DotNode{
		Name: n.Name(),
		Attrs: map[string]string{
			"label":     "boundary closer",
		},
	}
}
