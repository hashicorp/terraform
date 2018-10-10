package pg

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
)

func TestRemoteClient_impl(t *testing.T) {
	var _ remote.Client = new(RemoteClient)
	var _ remote.ClientLocker = new(RemoteClient)
}

func TestRemoteClient(t *testing.T) {
	connStr := getDatabaseUrl()
	schemaName := fmt.Sprintf("terraform_%s", t.Name())
	dbCleaner, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatal(err)
	}
	defer dbCleaner.Query(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))

	config := map[string]interface{}{
		"conn_str":    connStr,
		"schema_name": schemaName,
	}
	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	if b == nil {
		t.Fatal("Backend could not be configured")
	}

	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestClient(t, s.(*remote.State).Client)
}

func TestRemoteLocks(t *testing.T) {
	connStr := getDatabaseUrl()
	schemaName := fmt.Sprintf("terraform_%s", t.Name())
	dbCleaner, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatal(err)
	}
	defer dbCleaner.Query(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))

	config := map[string]interface{}{
		"conn_str":    connStr,
		"schema_name": schemaName,
	}
	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	remote.TestRemoteLocks(t, s.(*remote.State).Client, s.(*remote.State).Client)
}
