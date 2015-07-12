package terraform

import (
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
)

func TestProxyTransformer(t *testing.T) {
	var g Graph
	proxy := &testNodeProxy{NameValue: "proxy"}
	g.Add("A")
	g.Add("C")
	g.Add(proxy)
	g.Connect(dag.BasicEdge("A", proxy))
	g.Connect(dag.BasicEdge(proxy, "C"))

	{
		tf := &ProxyTransformer{}
		if err := tf.Transform(&g); err != nil {
			t.Fatalf("err: %s", err)
		}
	}

	actual := strings.TrimSpace(g.String())
	expected := strings.TrimSpace(testProxyTransformStr)
	if actual != expected {
		t.Fatalf("bad: %s", actual)
	}
}

type testNodeProxy struct {
	NameValue string
}

func (n *testNodeProxy) Name() string {
	return n.NameValue
}

func (n *testNodeProxy) Proxy() bool {
	return true
}

const testProxyTransformStr = `
A
  C
  proxy
C
proxy
  C
`
