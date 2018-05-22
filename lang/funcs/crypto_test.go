package funcs

import (
	"testing"
)

func TestUUID(t *testing.T) {
	result, err := UUID()
	if err != nil {
		t.Fatal(err)
	}

	resultStr := result.AsString()
	if got, want := len(resultStr), 36; got != want {
		t.Errorf("wrong result length %d; want %d", got, want)
	}
}
