package helper

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hashicorp/go-oracle-terraform/opc"
)

const TestEnvVar = "ORACLE_ACC"

// Test suite helpers

type TestCase struct {
	// Fields to test stuff with
}

func Test(t TestT, c TestCase) {
	if os.Getenv(TestEnvVar) == "" {
		t.Skip(fmt.Sprintf("Acceptance tests skipped unless env '%s' is set", TestEnvVar))
		return
	}

	// Setup logging Output
	logWriter, err := opc.LogOutput()
	if err != nil {
		t.Error(fmt.Sprintf("Error setting up log writer: %s", err))
	}
	log.SetOutput(logWriter)
}

type TestT interface {
	Error(args ...interface{})
	Fatal(args ...interface{})
	Skip(args ...interface{})
}

func RInt() int {
	rand.Seed(time.Now().UTC().UnixNano())
	return rand.Int()
}
