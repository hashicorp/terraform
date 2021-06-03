package ssh

import (
	"reflect"
	"testing"
)

func TestPasswordKeybardInteractive_Challenge(t *testing.T) {
	p := PasswordKeyboardInteractive("foo")
	result, err := p("foo", "bar", []string{"one", "two"}, nil)
	if err != nil {
		t.Fatalf("err not nil: %s", err)
	}

	if !reflect.DeepEqual(result, []string{"foo", "foo"}) {
		t.Fatalf("invalid password: %#v", result)
	}
}
