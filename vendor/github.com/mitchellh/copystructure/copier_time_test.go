package copystructure

import (
	"testing"
	"time"
)

func TestTimeCopier(t *testing.T) {
	v := time.Now().UTC()
	result, err := timeCopier(v)
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	if result.(time.Time) != v {
		t.Fatalf("bad: %#v\n\n%#v", v, result)
	}
}
