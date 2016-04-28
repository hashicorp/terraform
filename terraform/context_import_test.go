package terraform

import (
	"strings"
	"testing"
)

func TestContextImport_basic(t *testing.T) {
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

func TestContextImport_refresh(t *testing.T) {
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

	p.ImportStateReturn = []*InstanceState{
		&InstanceState{
			ID:        "foo",
			Ephemeral: EphemeralState{Type: "aws_instance"},
		},
	}

	p.RefreshFn = func(info *InstanceInfo, s *InstanceState) (*InstanceState, error) {
		return &InstanceState{
			ID:         "foo",
			Attributes: map[string]string{"foo": "bar"},
		}, nil
	}

	state, err := ctx.Import(&ImportOpts{
		Targets: []*ImportTarget{
			&ImportTarget{
				Addr: "aws_instance.foo",
				ID:   "bar",
			},
		},
	})
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	actual := strings.TrimSpace(state.String())
	expected := strings.TrimSpace(testImportRefreshStr)
	if actual != expected {
		t.Fatalf("bad: \n%s", actual)
	}
}

const testImportStr = `
aws_instance.foo:
  ID = foo
  provider = aws
`

const testImportRefreshStr = `
aws_instance.foo:
  ID = foo
  provider = aws
  foo = bar
`
