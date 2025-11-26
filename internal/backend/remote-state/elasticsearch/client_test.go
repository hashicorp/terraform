// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: BUSL-1.1

package elasticsearch

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/internal/backend"
	"github.com/hashicorp/terraform/internal/states/remote"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
	var _ remote.ClientLocker = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	testACC(t)

	endpoint := getEndpoint()

	config := backend.TestWrapConfig(map[string]interface{}{
		"endpoints":              []interface{}{endpoint},
		"index":                  fmt.Sprintf("terraform-test-%s", strings.ToLower(t.Name())),
		"username":               "elastic",
		"password":               "changeme",
		"skip_cert_verification": true,
	})

	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	if b == nil {
		t.Fatal("Backend could not be configured")
	}

	s, sDiags := b.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatal(sDiags)
	}

	remote.TestClient(t, s.(*remote.State).Client)
}

func TestRemoteLocks(t *testing.T) {
	testACC(t)

	endpoint := getEndpoint()

	config := backend.TestWrapConfig(map[string]interface{}{
		"endpoints":              []interface{}{endpoint},
		"index":                  fmt.Sprintf("terraform-test-%s", strings.ToLower(t.Name())),
		"username":               "elastic",
		"password":               "changeme",
		"skip_cert_verification": true,
	})

	b1 := backend.TestBackendConfig(t, New(), config).(*Backend)
	s1, sDiags := b1.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatal(sDiags)
	}

	b2 := backend.TestBackendConfig(t, New(), config).(*Backend)
	s2, sDiags := b2.StateMgr(backend.DefaultStateName)
	if sDiags.HasErrors() {
		t.Fatal(sDiags)
	}

	remote.TestRemoteLocks(t, s1.(*remote.State).Client, s2.(*remote.State).Client)
}
