package inmem

import (
	"flag"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/hashicorp/hcl2/hcl"

	"github.com/hashicorp/terraform/backend"
	"github.com/hashicorp/terraform/helper/logging"
	"github.com/hashicorp/terraform/state/remote"
	statespkg "github.com/hashicorp/terraform/states"
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

func TestBackend_impl(t *testing.T) {
	var _ backend.Backend = new(Backend)
}

func TestBackendConfig(t *testing.T) {
	defer Reset()
	testID := "test_lock_id"

	config := map[string]interface{}{
		"lock_id": testID,
	}

	b := backend.TestBackendConfig(t, New(), backend.TestWrapConfig(config)).(*Backend)

	s, err := b.StateMgr(backend.DefaultStateName)
	if err != nil {
		t.Fatal(err)
	}

	c := s.(*remote.State).Client.(*RemoteClient)
	if c.Name != backend.DefaultStateName {
		t.Fatal("client name is not configured")
	}

	if err := locks.unlock(backend.DefaultStateName, testID); err != nil {
		t.Fatalf("default state should have been locked: %s", err)
	}
}

func TestBackend(t *testing.T) {
	t.Fatalf("FIXME: this test seems to be hanging or getting into an infinite loop")
	defer Reset()
	b := backend.TestBackendConfig(t, New(), hcl.EmptyBody()).(*Backend)
	backend.TestBackendStates(t, b)
}

func TestBackendLocked(t *testing.T) {
	defer Reset()
	b1 := backend.TestBackendConfig(t, New(), hcl.EmptyBody()).(*Backend)
	b2 := backend.TestBackendConfig(t, New(), hcl.EmptyBody()).(*Backend)

	backend.TestBackendStateLocks(t, b1, b2)
}

// use the this backen to test the remote.State implementation
func TestRemoteState(t *testing.T) {
	t.Fatalf("FIXME: this test seems to be hanging or getting into an infinite loop")
	defer Reset()
	b := backend.TestBackendConfig(t, New(), hcl.EmptyBody())

	workspace := "workspace"

	// create a new workspace in this backend
	s, err := b.StateMgr(workspace)
	if err != nil {
		t.Fatal(err)
	}

	// force overwriting the remote state
	newState := statespkg.NewState()

	if err := s.WriteState(newState); err != nil {
		t.Fatal(err)
	}

	if err := s.PersistState(); err != nil {
		t.Fatal(err)
	}

	if err := s.RefreshState(); err != nil {
		t.Fatal(err)
	}
}
