package terraform

import (
	"strings"
	"testing"
)

func TestContextImport(t *testing.T) {
	p := testProvider("aws")
	ctx := testContext2(t, &ContextOpts{
		Providers: map[string]ResourceProviderFactory{
			"aws": testProviderFuncFixed(p),
		},
	})

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

const testImportStr = ``
