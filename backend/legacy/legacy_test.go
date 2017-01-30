package legacy

import (
	"testing"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/state/remote"
)

func TestInit(t *testing.T) {
	m := make(map[string]func() backend.Backend)
	Init(m)

	for k, _ := range remote.BuiltinClients {
		b, ok := m[k]
		if !ok {
			t.Fatalf("missing: %s", k)
		}

		if typ := b().(*Backend).Type; typ != k {
			t.Fatalf("bad type: %s", typ)
		}
	}
}

func TestInit_ignoreExisting(t *testing.T) {
	m := make(map[string]func() backend.Backend)
	m["local"] = nil
	Init(m)

	if v, ok := m["local"]; !ok || v != nil {
		t.Fatalf("bad: %#v", m)
	}
}
