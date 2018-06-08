package init

import (
	"testing"

	"github.com/hashicorp/terraform/backend/remote-state/inmem"
	"github.com/hashicorp/terraform/terraform"
)

func TestDeprecateBackend(t *testing.T) {
	deprecateMessage := "deprecated backend"
	deprecatedBackend := deprecateBackend(
		inmem.New(),
		deprecateMessage,
	)()

	warns, errs := deprecatedBackend.Validate(&terraform.ResourceConfig{})
	if errs != nil {
		for _, err := range errs {
			t.Error(err)
		}
		t.Fatal("validation errors")
	}

	if len(warns) != 1 {
		t.Fatalf("expected 1 warning, got %q", warns)
	}

	if warns[0] != deprecateMessage {
		t.Fatalf("expected %q, got %q", deprecateMessage, warns[0])
	}
}
