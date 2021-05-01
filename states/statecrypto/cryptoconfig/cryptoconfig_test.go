package cryptoconfig

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
)

func resetLogFatalf() {
	logFatalf = log.Fatalf
}

func TestBlankConfigurationProducesNoErrors(t *testing.T) {
	logFatalf = func(format string, v ...interface{}) {
		t.Errorf("received unexpected error: "+format, v...)
	}
	defer resetLogFatalf()

	_ = Configuration()
	_ = FallbackConfiguration()
}

func TestUnexpectedJsonInConfigurationProducesError(t *testing.T) {
	lastError := ""
	logFatalf = func(format string, v ...interface{}) {
		lastError = fmt.Sprintf(format, v...)
	}
	defer resetLogFatalf()

	envName := "TEST_CRYPTOCONFIG_TestInvalidJsonInConfigurationProducesError"
	configInvalid := `{"implementation":"something", "unexpectedField":"another thing"}`
	_ = os.Setenv(envName, configInvalid)
	defer os.Unsetenv(envName)

	_ = configFromEnv(envName)

	expected := "error parsing remote state encryption configuration from environment variable TEST_CRYPTOCONFIG_TestInvalidJsonInConfigurationProducesError: "
	if !strings.HasPrefix(lastError, expected) {
		t.Error("did not receive expected error")
	}
}
