// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package oras

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	cleanhttp "github.com/hashicorp/go-cleanhttp"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"

	svchost "github.com/hashicorp/terraform-svchost"
	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/backend/backendbase"
	"github.com/hashicorp/terraform/internal/command/cliconfig"
	"github.com/hashicorp/terraform/internal/configs/configschema"
	"github.com/hashicorp/terraform/internal/httpclient"
	pluginDiscovery "github.com/hashicorp/terraform/internal/plugin/discovery"
	"github.com/hashicorp/terraform/internal/states/remote"
	"github.com/hashicorp/terraform/internal/states/statemgr"
	"github.com/hashicorp/terraform/internal/tfdiags"
	"github.com/hashicorp/terraform/version"
	"golang.org/x/time/rate"
	orasRemote "oras.land/oras-go/v2/registry/remote"
	orasAuth "oras.land/oras-go/v2/registry/remote/auth"
	orasCredentials "oras.land/oras-go/v2/registry/remote/credentials"
)

const envVarRepository = "TF_BACKEND_ORAS_REPOSITORY"

const (
	envVarRetryMax     = "TF_BACKEND_ORAS_RETRY_MAX"
	envVarRetryWaitMin = "TF_BACKEND_ORAS_RETRY_WAIT_MIN"
	envVarRetryWaitMax = "TF_BACKEND_ORAS_RETRY_WAIT_MAX"
	envVarLockTTL      = "TF_BACKEND_ORAS_LOCK_TTL"
	envVarRateLimit    = "TF_BACKEND_ORAS_RATE_LIMIT"
	envVarRateBurst    = "TF_BACKEND_ORAS_RATE_LIMIT_BURST"
)

type Backend struct {
	Base backendbase.Base

	repository  string
	insecure    bool
	caFile      string
	compression string
	lockTTL     time.Duration
	rateLimit   int
	rateBurst   int
	retryCfg    RetryConfig

	versioningEnabled     bool
	versioningMaxVersions int

	repoClient *orasRepositoryClient
}

func New() backend.Backend {
	return &Backend{
		Base: backendbase.Base{
			Schema: &configschema.Block{
				Attributes: map[string]*configschema.Attribute{
					"repository": {
						Type:        cty.String,
						Optional:    true, // required via SDKLikeDefaults.Required
						Description: "OCI repository in the form <registry>/<repository>, without tag or digest. Can also be set via TF_BACKEND_ORAS_REPOSITORY env var.",
					},
					"insecure": {
						Type:        cty.Bool,
						Optional:    true,
						Description: "Skip TLS certificate verification when communicating with the OCI registry.",
					},
					"ca_file": {
						Type:        cty.String,
						Optional:    true,
						Description: "Path to a PEM-encoded CA certificate bundle to trust when communicating with the OCI registry.",
					},
					"retry_max": {
						Type:        cty.Number,
						Optional:    true,
						Description: "The number of retries for transient registry requests.",
					},
					"retry_wait_min": {
						Type:        cty.Number,
						Optional:    true,
						Description: "The minimum time in seconds to wait between transient registry request attempts.",
					},
					"retry_wait_max": {
						Type:        cty.Number,
						Optional:    true,
						Description: "The maximum time in seconds to wait between transient registry request attempts.",
					},
					"compression": {
						Type:        cty.String,
						Optional:    true,
						Description: "State compression. Supported values: none, gzip.",
					},
					"lock_ttl": {
						Type:        cty.Number,
						Optional:    true,
						Description: "Lock TTL in seconds. If set, stale locks older than this will be automatically cleared. 0 disables.",
					},
					"rate_limit": {
						Type:        cty.Number,
						Optional:    true,
						Description: "Maximum registry requests per second. 0 disables rate limiting.",
					},
					"rate_limit_burst": {
						Type:        cty.Number,
						Optional:    true,
						Description: "Maximum burst size for rate limiting. 0 uses default burst of 1.",
					},
				},
				BlockTypes: map[string]*configschema.NestedBlock{
					"versioning": {
						Nesting: configschema.NestingSingle,
						Block: configschema.Block{
							Attributes: map[string]*configschema.Attribute{
								"enabled": {
									Type:        cty.Bool,
									Optional:    true,
									Description: "Enable state versioning.",
								},
								"max_versions": {
									Type:        cty.Number,
									Optional:    true,
									Description: "Maximum number of historical versions to keep. 0 means unlimited.",
								},
							},
						},
					},
				},
			},
			SDKLikeDefaults: backendbase.SDKLikeDefaults{
				"repository":       {EnvVars: []string{envVarRepository}, Required: true},
				"retry_max":        {EnvVars: []string{envVarRetryMax}, Fallback: "2"},
				"retry_wait_min":   {EnvVars: []string{envVarRetryWaitMin}, Fallback: "1"},
				"retry_wait_max":   {EnvVars: []string{envVarRetryWaitMax}, Fallback: "30"},
				"compression":      {Fallback: "none"},
				"lock_ttl":         {EnvVars: []string{envVarLockTTL}, Fallback: "0"},
				"rate_limit":       {EnvVars: []string{envVarRateLimit}, Fallback: "0"},
				"rate_limit_burst": {EnvVars: []string{envVarRateBurst}, Fallback: "0"},
			},
		},
		retryCfg:    DefaultRetryConfig(),
		compression: "none",
	}
}

func (b *Backend) ConfigSchema() *configschema.Block {
	return b.Base.ConfigSchema()
}

func (b *Backend) PrepareConfig(configVal cty.Value) (cty.Value, tfdiags.Diagnostics) {
	return b.Base.PrepareConfig(configVal)
}

func (b *Backend) Configure(configVal cty.Value) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	b.repository = configVal.GetAttr("repository").AsString()
	b.insecure = !configVal.GetAttr("insecure").IsNull() && configVal.GetAttr("insecure").True()
	b.caFile = ""
	if v := configVal.GetAttr("ca_file"); !v.IsNull() {
		b.caFile = v.AsString()
	}

	if b.repository == "" {
		return diags.Append(fmt.Errorf("repository must be set"))
	}

	if v := configVal.GetAttr("compression"); !v.IsNull() {
		b.compression = v.AsString()
	}
	switch b.compression {
	case "none", "gzip":
		// ok
	default:
		return diags.Append(fmt.Errorf("unsupported compression %q (supported: none, gzip)", b.compression))
	}

	var lockTTLSeconds int
	if err := gocty.FromCtyValue(configVal.GetAttr("lock_ttl"), &lockTTLSeconds); err != nil {
		return diags.Append(fmt.Errorf("invalid lock_ttl: %w", err))
	}
	if lockTTLSeconds < 0 {
		return diags.Append(fmt.Errorf("lock_ttl must be non-negative"))
	}
	b.lockTTL = time.Duration(lockTTLSeconds) * time.Second

	var rateLimit int
	var rateBurst int
	if err := gocty.FromCtyValue(configVal.GetAttr("rate_limit"), &rateLimit); err != nil {
		return diags.Append(fmt.Errorf("invalid rate_limit: %w", err))
	}
	if err := gocty.FromCtyValue(configVal.GetAttr("rate_limit_burst"), &rateBurst); err != nil {
		return diags.Append(fmt.Errorf("invalid rate_limit_burst: %w", err))
	}
	if rateLimit < 0 {
		return diags.Append(fmt.Errorf("rate_limit must be non-negative"))
	}
	if rateBurst < 0 {
		return diags.Append(fmt.Errorf("rate_limit_burst must be non-negative"))
	}
	b.rateLimit = rateLimit
	b.rateBurst = rateBurst

	var retryMax int
	var retryWaitMinSeconds int
	var retryWaitMaxSeconds int
	if err := gocty.FromCtyValue(configVal.GetAttr("retry_max"), &retryMax); err != nil {
		return diags.Append(fmt.Errorf("invalid retry_max: %w", err))
	}
	if err := gocty.FromCtyValue(configVal.GetAttr("retry_wait_min"), &retryWaitMinSeconds); err != nil {
		return diags.Append(fmt.Errorf("invalid retry_wait_min: %w", err))
	}
	if err := gocty.FromCtyValue(configVal.GetAttr("retry_wait_max"), &retryWaitMaxSeconds); err != nil {
		return diags.Append(fmt.Errorf("invalid retry_wait_max: %w", err))
	}

	retryCfg := DefaultRetryConfig()
	retryCfg.MaxAttempts = retryMax + 1
	retryCfg.InitialBackoff = time.Duration(retryWaitMinSeconds) * time.Second
	retryCfg.MaxBackoff = time.Duration(retryWaitMaxSeconds) * time.Second
	if retryCfg.MaxAttempts < 1 {
		retryCfg.MaxAttempts = 1
	}
	if retryCfg.InitialBackoff <= 0 {
		retryCfg.InitialBackoff = time.Second
	}
	if retryCfg.MaxBackoff > 0 && retryCfg.MaxBackoff < retryCfg.InitialBackoff {
		retryCfg.MaxBackoff = retryCfg.InitialBackoff
	}
	b.retryCfg = retryCfg

	b.versioningEnabled = false
	b.versioningMaxVersions = 0
	if v := configVal.GetAttr("versioning"); !v.IsNull() {
		enabled := v.GetAttr("enabled")
		if !enabled.IsNull() {
			b.versioningEnabled = enabled.True()
		}
		var maxVersions int
		if err := gocty.FromCtyValue(v.GetAttr("max_versions"), &maxVersions); err != nil {
			return diags.Append(fmt.Errorf("invalid versioning.max_versions: %w", err))
		}
		b.versioningMaxVersions = maxVersions
	}
	if b.versioningEnabled && b.versioningMaxVersions < 0 {
		b.versioningMaxVersions = 0
	}

	repoClient, err := newORASRepositoryClient(b.repository, b.insecure, b.caFile, b.rateLimit, b.rateBurst)
	if err != nil {
		return diags.Append(err)
	}
	b.repoClient = repoClient

	return diags
}

func (b *Backend) getRepository() (*orasRepositoryClient, error) {
	if b.repoClient == nil {
		return nil, fmt.Errorf("backend is not configured")
	}
	return b.repoClient, nil
}

type orasRepositoryClient struct {
	repository string
	credFn     func(ctx context.Context, host string) (orasAuth.Credential, error)
	inner      orasRepository
}

func (r *orasRepositoryClient) accessTokenForHost(ctx context.Context, host string) (string, error) {
	if r == nil || r.credFn == nil {
		return "", nil
	}
	cred, err := r.credFn(ctx, host)
	if err != nil {
		return "", err
	}
	if cred.AccessToken != "" {
		return cred.AccessToken, nil
	}
	// Docker-style credentials often come back as user/password, where password
	// is the token (e.g. PAT). Use it as a best-effort bearer token.
	if cred.Password != "" {
		return cred.Password, nil
	}
	return "", nil
}

type orasRepository interface {
	Push(ctx context.Context, expected ocispec.Descriptor, content io.Reader) error
	Fetch(ctx context.Context, target ocispec.Descriptor) (io.ReadCloser, error)
	Resolve(ctx context.Context, reference string) (ocispec.Descriptor, error)
	Tag(ctx context.Context, desc ocispec.Descriptor, reference string) error
	Delete(ctx context.Context, target ocispec.Descriptor) error
	Exists(ctx context.Context, target ocispec.Descriptor) (bool, error)
	Tags(ctx context.Context, last string, fn func(tags []string) error) error
}

func newORASRepositoryClient(repository string, insecure bool, caFile string, rateLimit int, rateBurst int) (*orasRepositoryClient, error) {
	repo, err := orasRemote.NewRepository(repository)
	if err != nil {
		return nil, fmt.Errorf("invalid OCI repository %q: %w", repository, err)
	}

	httpClient, err := newORASHTTPClient(insecure, caFile, rateLimit, rateBurst)
	if err != nil {
		return nil, err
	}

	credFn := combinedCredentialFunc()
	repo.Client = &orasAuth.Client{
		Client:     httpClient,
		Credential: credFn,
	}

	return &orasRepositoryClient{repository: repository, credFn: credFn, inner: repo}, nil
}

func newORASHTTPClient(insecure bool, caFile string, rateLimit int, rateBurst int) (*http.Client, error) {
	client := cleanhttp.DefaultPooledClient()

	if t, ok := client.Transport.(*http.Transport); ok {
		t = t.Clone()
		if t.TLSClientConfig == nil {
			t.TLSClientConfig = &tls.Config{}
		}
		t.TLSClientConfig.InsecureSkipVerify = insecure
		if caFile != "" {
			pem, err := os.ReadFile(caFile)
			if err != nil {
				return nil, fmt.Errorf("reading ca_file %q: %w", caFile, err)
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(pem) {
				return nil, fmt.Errorf("ca_file %q: no valid certificates", caFile)
			}
			t.TLSClientConfig.RootCAs = pool
		}
		client.Transport = t
	}

	var limiter requestLimiter
	if rateLimit > 0 {
		if rateBurst <= 0 {
			rateBurst = 1
		}
		limiter = rate.NewLimiter(rate.Limit(rateLimit), rateBurst)
	}

	var rt http.RoundTripper = &userAgentRoundTripper{userAgent: httpclient.TerraformUserAgent(version.Version), inner: client.Transport}
	if limiter != nil {
		rt = &rateLimitedRoundTripper{limiter: limiter, inner: rt}
	}
	client.Transport = rt

	return client, nil
}

type userAgentRoundTripper struct {
	userAgent string
	inner     http.RoundTripper
}

func (rt *userAgentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", rt.userAgent)
	}
	return rt.inner.RoundTrip(req)
}

type requestLimiter interface {
	Wait(ctx context.Context) error
}

type rateLimitedRoundTripper struct {
	limiter requestLimiter
	inner   http.RoundTripper
}

func (rt *rateLimitedRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.limiter != nil {
		if err := rt.limiter.Wait(req.Context()); err != nil {
			return nil, err
		}
	}
	return rt.inner.RoundTrip(req)
}

func (b *Backend) StateMgr(workspace string) (statemgr.Full, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	repo, err := b.getRepository()
	if err != nil {
		return nil, diags.Append(err)
	}
	client := newRemoteClient(repo, workspace)
	client.retryConfig = b.retryCfg
	client.versioningEnabled = b.versioningEnabled
	client.versioningMaxVersions = b.versioningMaxVersions
	client.stateCompression = b.compression
	client.lockTTL = b.lockTTL

	return &remote.State{Client: client}, diags
}

func (b *Backend) Workspaces() ([]string, tfdiags.Diagnostics) {
	var diags tfdiags.Diagnostics

	repo, err := b.getRepository()
	if err != nil {
		return nil, diags.Append(err)
	}
	wss, err := listWorkspacesFromTags(repo)
	if err != nil {
		if isNotFound(err) {
			return []string{backend.DefaultStateName}, diags
		}
		return nil, diags.Append(err)
	}
	if len(wss) == 0 {
		return []string{backend.DefaultStateName}, diags
	}
	return wss, diags
}

func (b *Backend) DeleteWorkspace(name string, _ bool) tfdiags.Diagnostics {
	var diags tfdiags.Diagnostics

	if name == backend.DefaultStateName || name == "" {
		return diags.Append(fmt.Errorf("can't delete default state"))
	}

	repo, err := b.getRepository()
	if err != nil {
		return diags.Append(err)
	}

	wsTag := workspaceTagFor(name)
	stateRef := stateTagPrefix + wsTag
	lockRef := lockTagPrefix + wsTag
	stateVersionPrefix := stateRef + stateVersionTagSeparator

	// Best-effort cleanup - we ignore errors since some registries don't support DELETE
	ctx := context.Background()
	if desc, err := repo.inner.Resolve(ctx, stateRef); err == nil {
		_ = repo.inner.Delete(ctx, desc)
	}
	_ = repo.inner.Tags(ctx, "", func(page []string) error {
		for _, tag := range page {
			if !strings.HasPrefix(tag, stateVersionPrefix) {
				continue
			}
			if desc, err := repo.inner.Resolve(ctx, tag); err == nil {
				_ = repo.inner.Delete(ctx, desc)
			}
		}
		return nil
	})
	if desc, err := repo.inner.Resolve(ctx, lockRef); err == nil {
		_ = repo.inner.Delete(ctx, desc)
	}
	return diags
}

var (
	tfCredsOnce sync.Once
	tfCredsSrc  *cliconfig.CredentialsSource
	tfCredsErr  error
)

func terraformCredentialsSource() (*cliconfig.CredentialsSource, error) {
	tfCredsOnce.Do(func() {
		cfg, diags := cliconfig.LoadConfig()
		if diags.HasErrors() {
			tfCredsErr = diags.Err()
			return
		}
		src, err := cfg.CredentialsSource(pluginDiscovery.PluginMetaSet{})
		if err != nil {
			tfCredsErr = err
			return
		}
		tfCredsSrc = src
	})
	return tfCredsSrc, tfCredsErr
}

func terraformTokenCredentialFunc() func(ctx context.Context, host string) (orasAuth.Credential, error) {
	return func(ctx context.Context, host string) (orasAuth.Credential, error) {
		_ = ctx

		src, err := terraformCredentialsSource()
		if err != nil {
			return orasAuth.EmptyCredential, err
		}
		if src == nil {
			return orasAuth.EmptyCredential, nil
		}

		cmpHost, err := svchost.ForComparison(svchost.ForDisplay(host))
		if err != nil {
			return orasAuth.EmptyCredential, fmt.Errorf("invalid registry hostname %q: %w", host, err)
		}

		creds, err := src.ForHost(cmpHost)
		if err != nil {
			return orasAuth.EmptyCredential, err
		}
		if creds == nil || creds.Token() == "" {
			return orasAuth.EmptyCredential, nil
		}

		// Terraform host credentials are token-only today.
		// We map that token to an ORAS access token.
		return orasAuth.Credential{AccessToken: creds.Token()}, nil
	}
}

func dockerCredentialFunc() func(ctx context.Context, host string) (orasAuth.Credential, error) {
	store, err := orasCredentials.NewStoreFromDocker(orasCredentials.StoreOptions{})
	if err != nil {
		// Docker config isn't always present; treat as "no credentials".
		return func(context.Context, string) (orasAuth.Credential, error) {
			return orasAuth.EmptyCredential, nil
		}
	}
	return orasCredentials.Credential(store)
}

// combinedCredentialFunc tries docker creds first (most common), then terraform login tokens.
func combinedCredentialFunc() func(ctx context.Context, host string) (orasAuth.Credential, error) {
	dockerFn := dockerCredentialFunc()
	tfFn := terraformTokenCredentialFunc()

	return func(ctx context.Context, host string) (orasAuth.Credential, error) {
		if dockerFn != nil {
			cred, err := dockerFn(ctx, host)
			if err == nil && cred != orasAuth.EmptyCredential {
				return cred, nil
			}
			// If dockerFn errored, fall through to Terraform token as a best effort.
		}

		cred, err := tfFn(ctx, host)
		if err != nil {
			return orasAuth.EmptyCredential, err
		}
		if cred != orasAuth.EmptyCredential {
			return cred, nil
		}

		return orasAuth.EmptyCredential, nil
	}
}
