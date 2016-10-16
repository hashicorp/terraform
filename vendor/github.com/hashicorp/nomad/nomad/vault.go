package nomad

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/nomad/structs/config"
	vapi "github.com/hashicorp/vault/api"
	"github.com/mitchellh/mapstructure"
)

const (
	// vaultTokenCreateTTL is the duration the wrapped token for the client is
	// valid for. The units are in seconds.
	vaultTokenCreateTTL = "60s"

	// minimumTokenTTL is the minimum Token TTL allowed for child tokens.
	minimumTokenTTL = 5 * time.Minute
)

// VaultClient is the Servers interface for interfacing with Vault
type VaultClient interface {
	// CreateToken takes an allocation and task and returns an appropriate Vault
	// Secret
	CreateToken(a *structs.Allocation, task string) (*vapi.Secret, error)

	// LookupToken takes a token string and returns its capabilities.
	LookupToken(token string) (*vapi.Secret, error)

	// Stop is used to stop token renewal.
	Stop()
}

// tokenData holds the relevant information about the Vault token passed to the
// client.
type tokenData struct {
	CreationTTL int      `mapstructure:"creation_ttl"`
	TTL         int      `mapstructure:"ttl"`
	Renewable   bool     `mapstructure:"renewable"`
	Policies    []string `mapstructure:"policies"`
	Role        string   `mapstructure:"role"`
	Root        bool
}

// vaultClient is the Servers implementation of the VaultClient interface. The
// client renews the PeriodicToken given in the Vault configuration and provides
// the Server with the ability to create child tokens and lookup the permissions
// of tokens.
type vaultClient struct {
	// client is the Vault API client
	client *vapi.Client

	// auth is the Vault token auth API client
	auth *vapi.TokenAuth

	// config is the user passed Vault config
	config *config.VaultConfig

	// renewalRunning marks whether the renewal goroutine is running
	renewalRunning bool

	// establishingConn marks whether we are trying to establishe a connection to Vault
	establishingConn bool

	// connEstablished marks whether we have an established connection to Vault
	connEstablished bool

	// tokenData is the data of the passed Vault token
	token *tokenData

	// enabled indicates whether the vaultClient is enabled. If it is not the
	// token lookup and create methods will return errors.
	enabled bool

	// childTTL is the TTL for child tokens.
	childTTL string

	// lastRenewed is the time the token was last renewed
	lastRenewed time.Time

	shutdownCh chan struct{}
	l          sync.Mutex
	logger     *log.Logger
}

// NewVaultClient returns a Vault client from the given config. If the client
// couldn't be made an error is returned. If an error is not returned, Shutdown
// is expected to be called to clean up any created goroutine
func NewVaultClient(c *config.VaultConfig, logger *log.Logger) (*vaultClient, error) {
	if c == nil {
		return nil, fmt.Errorf("must pass valid VaultConfig")
	}

	if logger == nil {
		return nil, fmt.Errorf("must pass valid logger")
	}

	v := &vaultClient{
		enabled: c.Enabled,
		config:  c,
		logger:  logger,
	}

	// If vault is not enabled do not configure an API client or start any token
	// renewal.
	if !v.enabled {
		return v, nil
	}

	// Validate we have the required fields.
	if c.Token == "" {
		return nil, errors.New("Vault token must be set")
	} else if c.Addr == "" {
		return nil, errors.New("Vault address must be set")
	}

	// Parse the TTL if it is set
	if c.TaskTokenTTL != "" {
		d, err := time.ParseDuration(c.TaskTokenTTL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse TaskTokenTTL %q: %v", c.TaskTokenTTL, err)
		}

		if d.Nanoseconds() < minimumTokenTTL.Nanoseconds() {
			return nil, fmt.Errorf("ChildTokenTTL is less than minimum allowed of %v", minimumTokenTTL)
		}

		v.childTTL = c.TaskTokenTTL
	}

	// Get the Vault API configuration
	apiConf, err := c.ApiConfig()
	if err != nil {
		return nil, fmt.Errorf("Failed to create Vault API config: %v", err)
	}

	// Create the Vault API client
	client, err := vapi.NewClient(apiConf)
	if err != nil {
		v.logger.Printf("[ERR] vault: failed to create Vault client. Not retrying: %v", err)
		return nil, err
	}

	// Set the token and store the client
	client.SetToken(v.config.Token)
	v.client = client
	v.auth = client.Auth().Token()

	// Prepare and launch the token renewal goroutine
	v.shutdownCh = make(chan struct{})
	go v.establishConnection()
	return v, nil
}

// establishConnection is used to make first contact with Vault. This should be
// called in a go-routine since the connection is retried til the Vault Client
// is stopped or the connection is successfully made at which point the renew
// loop is started.
func (v *vaultClient) establishConnection() {
	v.l.Lock()
	v.establishingConn = true
	v.l.Unlock()

	// Create the retry timer and set initial duration to zero so it fires
	// immediately
	retryTimer := time.NewTimer(0)

OUTER:
	for {
		select {
		case <-v.shutdownCh:
			return
		case <-retryTimer.C:
			// Ensure the API is reachable
			if _, err := v.client.Sys().InitStatus(); err != nil {
				v.logger.Printf("[WARN] vault: failed to contact Vault API. Retrying in %v",
					v.config.ConnectionRetryIntv)
				retryTimer.Reset(v.config.ConnectionRetryIntv)
				continue OUTER
			}

			break OUTER
		}
	}

	v.l.Lock()
	v.connEstablished = true
	v.establishingConn = false
	v.l.Unlock()

	// Retrieve our token, validate it and parse the lease duration
	if err := v.parseSelfToken(); err != nil {
		v.logger.Printf("[ERR] vault: failed to lookup self token and not retrying: %v", err)
		return
	}

	// Set the wrapping function such that token creation is wrapped now
	// that we know our role
	v.client.SetWrappingLookupFunc(v.getWrappingFn())

	// If we are given a non-root token, start renewing it
	if v.token.Root {
		v.logger.Printf("[DEBUG] vault: not renewing token as it is root")
	} else {
		v.logger.Printf("[DEBUG] vault: token lease duration is %v",
			time.Duration(v.token.CreationTTL)*time.Second)
		go v.renewalLoop()
	}
}

// renewalLoop runs the renew loop. This should only be called if we are given a
// non-root token.
func (v *vaultClient) renewalLoop() {
	v.l.Lock()
	v.renewalRunning = true
	v.l.Unlock()

	// Create the renewal timer and set initial duration to zero so it fires
	// immediately
	authRenewTimer := time.NewTimer(0)

	// Backoff is to reduce the rate we try to renew with Vault under error
	// situations
	backoff := 0.0

	for {
		select {
		case <-v.shutdownCh:
			return
		case <-authRenewTimer.C:
			// Renew the token and determine the new expiration
			err := v.renew()
			currentExpiration := v.lastRenewed.Add(time.Duration(v.token.CreationTTL) * time.Second)

			// Successfully renewed
			if err == nil {
				// If we take the expiration (lastRenewed + auth duration) and
				// subtract the current time, we get a duration until expiry.
				// Set the timer to poke us after half of that time is up.
				durationUntilRenew := currentExpiration.Sub(time.Now()) / 2

				v.logger.Printf("[INFO] vault: renewing token in %v", durationUntilRenew)
				authRenewTimer.Reset(durationUntilRenew)

				// Reset any backoff
				backoff = 0
				break
			}

			// Back off, increasing the amount of backoff each time. There are some rules:
			//
			// * If we have an existing authentication that is going to expire,
			// never back off more than half of the amount of time remaining
			// until expiration
			// * Never back off more than 30 seconds multiplied by a random
			// value between 1 and 2
			// * Use randomness so that many clients won't keep hitting Vault
			// at the same time

			// Set base values and add some backoff

			v.logger.Printf("[DEBUG] vault: got error or bad auth, so backing off: %v", err)
			switch {
			case backoff < 5:
				backoff = 5
			case backoff >= 24:
				backoff = 30
			default:
				backoff = backoff * 1.25
			}

			// Add randomness
			backoff = backoff * (1.0 + rand.Float64())

			maxBackoff := currentExpiration.Sub(time.Now()) / 2
			if maxBackoff < 0 {
				// We have failed to renew the token past its expiration. Stop
				// renewing with Vault.
				v.l.Lock()
				defer v.l.Unlock()
				v.logger.Printf("[ERR] vault: failed to renew Vault token before lease expiration. Renew loop exiting")
				if v.renewalRunning {
					v.renewalRunning = false
					close(v.shutdownCh)
				}

				return

			} else if backoff > maxBackoff.Seconds() {
				backoff = maxBackoff.Seconds()
			}

			durationUntilRetry := time.Duration(backoff) * time.Second
			v.logger.Printf("[INFO] vault: backing off for %v", durationUntilRetry)

			authRenewTimer.Reset(durationUntilRetry)
		}
	}
}

// renew attempts to renew our Vault token. If the renewal fails, an error is
// returned. This method updates the lastRenewed time
func (v *vaultClient) renew() error {
	// Attempt to renew the token
	secret, err := v.auth.RenewSelf(v.token.CreationTTL)
	if err != nil {
		return err
	}

	auth := secret.Auth
	if auth == nil {
		return fmt.Errorf("renewal successful but not auth information returned")
	} else if auth.LeaseDuration == 0 {
		return fmt.Errorf("renewal successful but no lease duration returned")
	}

	v.lastRenewed = time.Now()
	v.logger.Printf("[DEBUG] vault: succesfully renewed server token")
	return nil
}

// getWrappingFn returns an appropriate wrapping function for Nomad Servers
func (v *vaultClient) getWrappingFn() func(operation, path string) string {
	createPath := "auth/token/create"
	if !v.token.Root {
		createPath = fmt.Sprintf("auth/token/create/%s", v.token.Role)
	}

	return func(operation, path string) string {
		// Only wrap the token create operation
		if operation != "POST" || path != createPath {
			return ""
		}

		return vaultTokenCreateTTL
	}
}

// parseSelfToken looks up the Vault token in Vault and parses its data storing
// it in the client. If the token is not valid for Nomads purposes an error is
// returned.
func (v *vaultClient) parseSelfToken() error {
	// Get the initial lease duration
	auth := v.client.Auth().Token()
	self, err := auth.LookupSelf()
	if err != nil {
		return fmt.Errorf("failed to lookup Vault periodic token: %v", err)
	}

	// Read and parse the fields
	var data tokenData
	if err := mapstructure.WeakDecode(self.Data, &data); err != nil {
		return fmt.Errorf("failed to parse Vault token's data block: %v", err)
	}

	root := false
	for _, p := range data.Policies {
		if p == "root" {
			root = true
			break
		}
	}

	if !data.Renewable && !root {
		return fmt.Errorf("Vault token is not renewable or root")
	}

	if data.CreationTTL == 0 && !root {
		return fmt.Errorf("invalid lease duration of zero")
	}

	if data.TTL == 0 && !root {
		return fmt.Errorf("token TTL is zero")
	}

	if !root && data.Role == "" {
		return fmt.Errorf("token role name must be set when not using a root token")
	}

	data.Root = root
	v.token = &data
	return nil
}

// Stop stops any goroutine that may be running, either for establishing a Vault
// connection or token renewal.
func (v *vaultClient) Stop() {
	// Nothing to do
	if !v.enabled {
		return
	}

	v.l.Lock()
	defer v.l.Unlock()
	if !v.renewalRunning || !v.establishingConn {
		return
	}

	close(v.shutdownCh)
	v.renewalRunning = false
	v.establishingConn = false
}

// ConnectionEstablished returns whether a connection to Vault has been
// established.
func (v *vaultClient) ConnectionEstablished() bool {
	v.l.Lock()
	defer v.l.Unlock()
	return v.connEstablished
}

func (v *vaultClient) CreateToken(a *structs.Allocation, task string) (*vapi.Secret, error) {
	return nil, nil
}

// LookupToken takes a Vault token and does a lookup against Vault
func (v *vaultClient) LookupToken(token string) (*vapi.Secret, error) {
	// Nothing to do
	if !v.enabled {
		return nil, fmt.Errorf("Vault integration disabled")
	}

	// Check if we have established a connection with Vault
	if !v.ConnectionEstablished() {
		return nil, fmt.Errorf("Connection to Vault has not been established. Retry")
	}

	// Lookup the token
	return v.auth.Lookup(token)
}

// PoliciesFrom parses the set of policies returned by a token lookup.
func PoliciesFrom(s *vapi.Secret) ([]string, error) {
	if s == nil {
		return nil, fmt.Errorf("cannot parse nil Vault secret")
	}
	var data tokenData
	if err := mapstructure.WeakDecode(s.Data, &data); err != nil {
		return nil, fmt.Errorf("failed to parse Vault token's data block: %v", err)
	}

	return data.Policies, nil
}
