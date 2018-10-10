package pg

// Create the test database: createdb terraform_backend_pg_test
// TF_PG_TEST=true make test TEST=./backend/remote-state/pg TESTARGS='-v -run ^TestBackend'

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
	_ "github.com/lib/pq"
)

// verify that we are doing ACC tests or the Postgres tests specifically
func testACC(t *testing.T) {
	skip := os.Getenv("TF_ACC") == "" && os.Getenv("TF_PG_TEST") == ""
	if skip {
		t.Log("pg backend tests require setting TF_ACC or TF_PG_TEST")
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

	config := map[string]interface{}{
		"conn_str":    connStr,
		"schema_name": schemaName,
	}
	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	if b == nil {
		t.Fatal("Backend could not be configured")
	}

	_, err = b.db.Query(fmt.Sprintf("SELECT name, info, created_at FROM %s.%s LIMIT 1", schemaName, locksTableName))
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.db.Query(fmt.Sprintf("SELECT name, data FROM %s.%s LIMIT 1", schemaName, statesTableName))
	if err != nil {
		t.Fatal(err)
	}

	_, err = b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	s, err := b.State(backend.DefaultStateName)
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

	config := map[string]interface{}{
		"conn_str":    connStr,
		"schema_name": schemaName,
	}
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

	config := map[string]interface{}{
		"conn_str":    connStr,
		"schema_name": schemaName,
	}
	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	if b == nil {
		t.Fatal("Backend could not be configured")
	}

	bb := backend.TestBackendConfig(t, New(), config).(*Backend)

	if bb == nil {
		t.Fatal("Backend could not be configured")
	}

	backend.TestBackendStateLocks(t, b, bb)
	backend.TestBackendStateForceUnlock(t, b, bb)
}

func TestBackendStateLocksDisabled(t *testing.T) {
	testACC(t)
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
		"lock":        false,
	}
	b := backend.TestBackendConfig(t, New(), config).(*Backend)

	if b == nil {
		t.Fatal("Backend could not be configured")
	}

	bb := backend.TestBackendConfig(t, New(), config).(*Backend)

	if bb == nil {
		t.Fatal("Backend could not be configured")
	}

	backend.TestBackendStates(t, b)
	backend.TestBackendStateLocks(t, b, bb)
}

func getDatabaseUrl() string {
	return os.Getenv("DATABASE_URL")
}
