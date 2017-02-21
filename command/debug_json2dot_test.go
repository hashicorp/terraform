package command

import (
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/hashicorp/terraform/dag"
	"github.com/mitchellh/cli"
)

func TestDebugJSON2Dot(t *testing.T) {
	// create the graph JSON output
	logFile, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(logFile.Name())

	var g dag.Graph
	g.SetDebugWriter(logFile)

	g.Add(1)
	g.Add(2)
	g.Add(3)
	g.Connect(dag.BasicEdge(1, 2))
	g.Connect(dag.BasicEdge(2, 3))

	ui := new(cli.MockUi)
	c := &DebugJSON2DotCommand{
		Meta: Meta{
			ContextOpts: testCtxConfig(testProvider()),
			Ui:          ui,
		},
	}

	args := []string{
		logFile.Name(),
	}
	if code := c.Run(args); code != 0 {
		t.Fatalf("bad: \n%s", ui.ErrorWriter.String())
	}

	output := ui.OutputWriter.String()
	if !strings.HasPrefix(output, "digraph {") {
		t.Fatalf("doesn't look like digraph: %s", output)
	}

	if !strings.Contains(output, `subgraph "root" {`) {
		t.Fatalf("doesn't contains root subgraph: %s", output)
	}
}
