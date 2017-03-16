package state

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/terraform/helper/logging"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if testing.Verbose() {
		// if we're verbose, use the logging requested by TF_LOG
		logging.SetOutput()
	} else {
		// otherwise silence all logs
		log.SetOutput(ioutil.Discard)
	}
	os.Exit(m.Run())
}

func TestNewLockInfo(t *testing.T) {
	info1 := NewLockInfo()
	info2 := NewLockInfo()

	if info1.ID == "" {
		t.Fatal("LockInfo missing ID")
	}

	if info1.Version == "" {
		t.Fatal("LockInfo missing version")
	}

	if info1.Created.IsZero() {
		t.Fatal("LockInfo missing Created")
	}

	if info1.ID == info2.ID {
		t.Fatal("multiple LockInfo with identical IDs")
	}

	// test the JSON output is valid
	newInfo := &LockInfo{}
	err := json.Unmarshal(info1.Marshal(), newInfo)
	if err != nil {
		t.Fatal(err)
	}
}
