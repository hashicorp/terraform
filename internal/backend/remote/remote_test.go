package remote

import (
	"flag"
	"os"
	"testing"
	"time"

	_ "github.com/hashicorp/terraform/internal/logging"
)

func TestMain(m *testing.M) {
	flag.Parse()

	// Make sure TF_FORCE_LOCAL_BACKEND is unset
	os.Unsetenv("TF_FORCE_LOCAL_BACKEND")

	// Reduce delays to make tests run faster
	backoffMin = 1.0
	backoffMax = 1.0
	planConfigurationVersionsPollInterval = 1 * time.Millisecond
	runPollInterval = 1 * time.Millisecond

	os.Exit(m.Run())
}
