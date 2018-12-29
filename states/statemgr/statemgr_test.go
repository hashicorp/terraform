package statemgr

import (
	"context"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"testing"
	"time"

	"github.com/hashicorp/terraform/helper/logging"
)

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

func TestLockWithContext(t *testing.T) {
	s := NewFullFake(nil, TestFullInitialState())

	id, err := s.Lock(NewLockInfo())
	if err != nil {
		t.Fatal(err)
	}

	// use a cancelled context for an immediate timeout
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	info := NewLockInfo()
	info.Info = "lock with context"
	_, err = LockWithContext(ctx, s, info)
	if err == nil {
		t.Fatal("lock should have failed immediately")
	}

	// block until LockwithContext has made a first attempt
	attempted := make(chan struct{})
	postLockHook = func() {
		close(attempted)
		postLockHook = nil
	}

	// unlock the state during LockWithContext
	unlocked := make(chan struct{})
	go func() {
		defer close(unlocked)
		<-attempted
		if err := s.Unlock(id); err != nil {
			t.Fatal(err)
		}
	}()

	ctx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	id, err = LockWithContext(ctx, s, info)
	if err != nil {
		t.Fatal("lock should have completed within 2s:", err)
	}

	// ensure the goruotine completes
	<-unlocked
}

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
