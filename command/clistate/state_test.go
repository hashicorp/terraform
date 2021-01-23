package clistate

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform/states/statemgr"
	"github.com/mitchellh/cli"
	"github.com/mitchellh/colorstring"
)

func TestUnlock(t *testing.T) {
	ui := new(cli.MockUi)

	l := NewLocker(context.Background(), 0, ui, &colorstring.Colorize{Disable: true})
	l.Lock(statemgr.NewUnlockErrorFull(nil, nil), "test-lock")

	err := l.Unlock(nil)
	if err != nil {
		t.Log(err.Error())
	} else {
		t.Error("expected error")
	}
}
