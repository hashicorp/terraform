package nomad

import (
	"encoding/json"
	"log"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/nomad/nomad/structs"
	"github.com/hashicorp/nomad/nomad/structs/config"
	"github.com/hashicorp/nomad/testutil"
	vapi "github.com/hashicorp/vault/api"
)

func TestVaultClient_BadConfig(t *testing.T) {
	conf := &config.VaultConfig{}
	logger := log.New(os.Stderr, "", log.LstdFlags)

	// Should be no error since Vault is not enabled
	client, err := NewVaultClient(conf, logger)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if client.ConnectionEstablished() {
		t.Fatalf("bad")
	}

	conf.Enabled = true
	_, err = NewVaultClient(conf, logger)
	if err == nil || !strings.Contains(err.Error(), "token must be set") {
		t.Fatalf("Expected token unset error: %v", err)
	}

	conf.Token = "123"
	_, err = NewVaultClient(conf, logger)
	if err == nil || !strings.Contains(err.Error(), "address must be set") {
		t.Fatalf("Expected address unset error: %v", err)
	}
}

// Test that the Vault Client can establish a connection even if it is started
// before Vault is available.
func TestVaultClient_EstablishConnection(t *testing.T) {
	v := testutil.NewTestVault(t)
	defer v.Stop()

	logger := log.New(os.Stderr, "", log.LstdFlags)
	v.Config.ConnectionRetryIntv = 100 * time.Millisecond
	client, err := NewVaultClient(v.Config, logger)
	if err != nil {
		t.Fatalf("failed to build vault client: %v", err)
	}
	defer client.Stop()

	// Sleep a little while and check that no connection has been established.
	time.Sleep(100 * time.Duration(testutil.TestMultiplier()) * time.Millisecond)

	if client.ConnectionEstablished() {
		t.Fatalf("ConnectionEstablished() returned true before Vault server started")
	}

	// Start Vault
	v.Start()

	waitForConnection(client, t)

	// Ensure that since we are using a root token that we haven started the
	// renewal loop.
	if client.renewalRunning {
		t.Fatalf("No renewal loop should be running")
	}
}

func TestVaultClient_RenewalLoop(t *testing.T) {
	v := testutil.NewTestVault(t).Start()
	defer v.Stop()

	// Build a role
	l := v.Client.Logical()
	d := make(map[string]interface{}, 2)
	d["allowed_policies"] = "default"
	d["period"] = 5
	l.Write("auth/token/roles/test", d)

	// Create a new token with the role
	a := v.Client.Auth().Token()
	req := vapi.TokenCreateRequest{}
	s, err := a.CreateWithRole(&req, "test")
	if err != nil {
		t.Fatalf("failed to create child token: %v", err)
	}

	// Get the client token
	if s == nil || s.Auth == nil {
		t.Fatalf("bad secret response: %+v", s)
	}

	// Set the configs token
	v.Config.Token = s.Auth.ClientToken

	// Start the client
	logger := log.New(os.Stderr, "", log.LstdFlags)
	client, err := NewVaultClient(v.Config, logger)
	if err != nil {
		t.Fatalf("failed to build vault client: %v", err)
	}
	defer client.Stop()

	// Sleep 8 seconds and ensure we have a non-zero TTL
	time.Sleep(8 * time.Second)

	// Get the current TTL
	s2, err := a.Lookup(v.Config.Token)
	if err != nil {
		t.Fatalf("failed to lookup token: %v", err)
	}

	ttl := parseTTLFromLookup(s2, t)
	if ttl == 0 {
		t.Fatalf("token renewal failed; ttl %v", ttl)
	}
}

func parseTTLFromLookup(s *vapi.Secret, t *testing.T) int64 {
	if s == nil {
		t.Fatalf("nil secret")
	} else if s.Data == nil {
		t.Fatalf("nil data block in secret")
	}

	ttlRaw, ok := s.Data["ttl"]
	if !ok {
		t.Fatalf("no ttl")
	}

	ttlNumber, ok := ttlRaw.(json.Number)
	if !ok {
		t.Fatalf("failed to convert ttl %q to json Number", ttlRaw)
	}

	ttl, err := ttlNumber.Int64()
	if err != nil {
		t.Fatalf("Failed to get ttl from json.Number: %v", err)
	}

	return ttl
}

func TestVaultClient_LookupToken_Invalid(t *testing.T) {
	conf := &config.VaultConfig{
		Enabled: false,
	}

	logger := log.New(os.Stderr, "", log.LstdFlags)
	client, err := NewVaultClient(conf, logger)
	if err != nil {
		t.Fatalf("failed to build vault client: %v", err)
	}

	_, err = client.LookupToken("foo")
	if err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("Expected error because Vault is disabled: %v", err)
	}

	// Enable vault but use a bad address so it never establishes a conn
	conf.Enabled = true
	conf.Addr = "http://foobar:12345"
	conf.Token = structs.GenerateUUID()
	client, err = NewVaultClient(conf, logger)
	if err != nil {
		t.Fatalf("failed to build vault client: %v", err)
	}

	_, err = client.LookupToken("foo")
	if err == nil || !strings.Contains(err.Error(), "established") {
		t.Fatalf("Expected error because connection to Vault hasn't been made: %v", err)
	}
}

func waitForConnection(v *vaultClient, t *testing.T) {
	testutil.WaitForResult(func() (bool, error) {
		return v.ConnectionEstablished(), nil
	}, func(err error) {
		t.Fatalf("Connection not established")
	})
}

func TestVaultClient_LookupToken(t *testing.T) {
	v := testutil.NewTestVault(t).Start()
	defer v.Stop()

	logger := log.New(os.Stderr, "", log.LstdFlags)
	client, err := NewVaultClient(v.Config, logger)
	if err != nil {
		t.Fatalf("failed to build vault client: %v", err)
	}

	waitForConnection(client, t)

	// Lookup ourselves
	s, err := client.LookupToken(v.Config.Token)
	if err != nil {
		t.Fatalf("self lookup failed: %v", err)
	}

	policies, err := PoliciesFrom(s)
	if err != nil {
		t.Fatalf("failed to parse policies: %v", err)
	}

	expected := []string{"root"}
	if !reflect.DeepEqual(policies, expected) {
		t.Fatalf("Unexpected policies; got %v; want %v", policies, expected)
	}

	// Create a token with a different set of policies
	expected = []string{"default"}
	req := vapi.TokenCreateRequest{
		Policies: expected,
	}
	s, err = v.Client.Auth().Token().Create(&req)
	if err != nil {
		t.Fatalf("failed to create child token: %v", err)
	}

	// Get the client token
	if s == nil || s.Auth == nil {
		t.Fatalf("bad secret response: %+v", s)
	}

	// Lookup new child
	s, err = client.LookupToken(s.Auth.ClientToken)
	if err != nil {
		t.Fatalf("self lookup failed: %v", err)
	}

	policies, err = PoliciesFrom(s)
	if err != nil {
		t.Fatalf("failed to parse policies: %v", err)
	}

	if !reflect.DeepEqual(policies, expected) {
		t.Fatalf("Unexpected policies; got %v; want %v", policies, expected)
	}
}
