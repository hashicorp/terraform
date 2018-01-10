package moduledeps

import (
	"testing"
)

func TestProviderInstance(t *testing.T) {
	tests := []struct {
		Name      string
		WantType  string
		WantAlias string
	}{
		{
			Name:      "aws",
			WantType:  "aws",
			WantAlias: "",
		},
		{
			Name:      "aws.foo",
			WantType:  "aws",
			WantAlias: "foo",
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			inst := ProviderInstance(test.Name)
			if got, want := inst.Type(), test.WantType; got != want {
				t.Errorf("got type %q; want %q", got, want)
			}
			if got, want := inst.Alias(), test.WantAlias; got != want {
				t.Errorf("got alias %q; want %q", got, want)
			}
		})
	}
}
