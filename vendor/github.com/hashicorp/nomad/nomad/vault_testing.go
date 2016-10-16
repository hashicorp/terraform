package nomad

import (
	"github.com/hashicorp/nomad/nomad/structs"
	vapi "github.com/hashicorp/vault/api"
)

// TestVaultClient is a Vault client appropriate for use during testing. Its
// behavior is programmable such that endpoints can be tested under various
// circumstances.
type TestVaultClient struct {
	// LookupTokenErrors maps a token to an error that will be returned by the
	// LookupToken call
	LookupTokenErrors map[string]error

	// LookupTokenSecret maps a token to the Vault secret that will be returned
	// by the LookupToken call
	LookupTokenSecret map[string]*vapi.Secret
}

func (v *TestVaultClient) LookupToken(token string) (*vapi.Secret, error) {
	var secret *vapi.Secret
	var err error

	if v.LookupTokenSecret != nil {
		secret = v.LookupTokenSecret[token]
	}
	if v.LookupTokenErrors != nil {
		err = v.LookupTokenErrors[token]
	}

	return secret, err
}

// SetLookupTokenSecret sets the error that will be returned by the token
// lookup
func (v *TestVaultClient) SetLookupTokenError(token string, err error) {
	if v.LookupTokenErrors == nil {
		v.LookupTokenErrors = make(map[string]error)
	}

	v.LookupTokenErrors[token] = err
}

// SetLookupTokenSecret sets the secret that will be returned by the token
// lookup
func (v *TestVaultClient) SetLookupTokenSecret(token string, secret *vapi.Secret) {
	if v.LookupTokenSecret == nil {
		v.LookupTokenSecret = make(map[string]*vapi.Secret)
	}

	v.LookupTokenSecret[token] = secret
}

// SetLookupTokenAllowedPolicies is a helper that adds a secret that allows the
// given policies
func (v *TestVaultClient) SetLookupTokenAllowedPolicies(token string, policies []string) {
	s := &vapi.Secret{
		Data: map[string]interface{}{
			"policies": policies,
		},
	}

	v.SetLookupTokenSecret(token, s)
}

func (v *TestVaultClient) CreateToken(a *structs.Allocation, task string) (*vapi.Secret, error) {
	return nil, nil
}

func (v *TestVaultClient) Stop() {}
