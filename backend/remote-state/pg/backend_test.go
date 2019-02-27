package pg

// Create the test database: createdb terraform_backend_pg_test
// TF_ACC=1 GO111MODULE=on go test -v -mod=vendor -timeout=2m -parallel=4 github.com/hashicorp/terraform/backend/remote-state/pg

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
	_ "github.com/lib/pq"
)

// Function to skip a test unless in ACCeptance test mode.
//
// A running Postgres server identified by env variable
// DATABASE_URL is required for acceptance tests.
func testACC(t *testing.T) {
	skip := os.Getenv("TF_ACC") == ""
	if skip {
		t.Log("pg backend tests require setting TF_ACC")
		t.Skip()
	}
	if os.Getenv("DATABASE_URL") == "" {
		os.Setenv("DATABASE_URL", "postgres://localhost/terraform_backend_pg_test?sslmode=disable")
	}
}

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackendConfig(t *testing.T) {
	testACC(t)
	connStr := getDatabaseUrl()
	schemaName := fmt.Sprintf("terraform_%s", t.Name())
	dbCleaner, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatal(err)
	}
	defer dbCleaner.Query(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))

	config := backend.TestWrapConfig(map[string]interface{}{
		"conn_str":    connStr,
		"schema_name": schemaName,
	})
	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	if b == nil {
		t.Fatal("Backend could not be configured")
	}

	_, err = b.db.Query(fmt.Sprintf("SELECT name, data FROM %s.%s LIMIT 1", schemaName, statesTableName))
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}
	c := s.(*remote.State).Client.(*RemoteClient)
	if c.Name != backend.DefaultStateName {
		t.Fatal("RemoteClient name is not configured")
	}
}

func TestBackendStates(t *testing.T) {
	testACC(t)
	connStr := getDatabaseUrl()
	schemaName := fmt.Sprintf("terraform_%s", t.Name())
	dbCleaner, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatal(err)
	}
	defer dbCleaner.Query(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))

	config := backend.TestWrapConfig(map[string]interface{}{
		"conn_str":    connStr,
		"schema_name": schemaName,
	})
	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	if b == nil {
		t.Fatal("Backend could not be configured")
	}

	backend.TestBackendStates(t, b)
}

func TestBackendStateLocks(t *testing.T) {
	testACC(t)
	connStr := getDatabaseUrl()
	schemaName := fmt.Sprintf("terraform_%s", t.Name())
	dbCleaner, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatal(err)
	}
	defer dbCleaner.Query(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))

	config := backend.TestWrapConfig(map[string]interface{}{
		"conn_str":    connStr,
		"schema_name": schemaName,
	})
	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	if b == nil {
		t.Fatal("Backend could not be configured")
	}

	bb := backend.TestBackendConfig(t, New(), config).(*Backend)

	if bb == nil {
		t.Fatal("Backend could not be configured")
	}

	backend.TestBackendStateLocks(t, b, bb)
}

func getDatabaseUrl() string {
	return os.Getenv("DATABASE_URL")
}
