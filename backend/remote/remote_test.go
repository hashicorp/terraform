package remote

import (
	"flag"
	"os"
	"testing"

	_ "github.com/hashicorp/terraform/internal/logging"
)

func TestMain(m *testing.M) {
	flag.Parse()

	// Make sure TF_FORCE_LOCAL_BACKEND is unset
	os.Unsetenv("TF_FORCE_LOCAL_BACKEND")

	os.Exit(m.Run())
}
