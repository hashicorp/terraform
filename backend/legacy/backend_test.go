package legacy

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/config"
	"github.com/hashicorp/terraform/state"
	"github.com/hashicorp/terraform/terraform"
)

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackend(t *testing.T) {
	td, err := ioutil.TempDir("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	defer os.RemoveAll(td)

	b := &Backend{Type: "local"}
	conf := terraform.NewResourceConfig(config.TestRawConfig(t, map[string]interface{}{
		"path": filepath.Join(td, "data"),
	}))

	// Config
	if err := b.Configure(conf); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Grab state
	s, err := b.State(backend.DefaultStateName)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if s == nil {
		t.Fatalf("state is nil")
	}

	// Test it
	s.WriteState(state.TestStateInitial())
	s.PersistState()
	state.TestState(t, s)
}
