package terraform

import (
	"fmt"
	"log"
	"strings"
)

// GraphBuilder is an interface that can be implemented and used with
// Terraform to build the graph that Terraform walks.
type GraphBuilder interface {
	// Build builds the graph for the given module path. It is up to
	// the interface implementation whether this build should expand
	// the graph or not.
	Build(path []string) (*Graph, error)
}

// BasicGraphBuilder is a GraphBuilder that builds a graph out of a
// series of transforms and (optionally) validates the graph is a valid
// structure.
type BasicGraphBuilder struct {
	Steps    []GraphTransformer
	Validate bool
	// Optional name to add to the graph debug log
	Name string
}

func (b *BasicGraphBuilder) Build(path []string) (*Graph, error) {
	g := &Graph{Path: path}

	debugName := "graph.json"
	if b.Name != "" {
		debugName = b.Name + "-" + debugName
	}
	debugBuf := dbug.NewFileWriter(debugName)
	g.SetDebugWriter(debugBuf)
	defer debugBuf.Close()

	for _, step := range b.Steps {
		if step == nil {
			continue
		}

		stepName := fmt.Sprintf("%T", step)
		dot := strings.LastIndex(stepName, ".")
		if dot >= 0 {
			stepName = stepName[dot+1:]
		}

		debugOp := g.DebugOperation(stepName, "")
		err := step.Transform(g)

		errMsg := ""
		if err != nil {
			errMsg = err.Error()
		}
		debugOp.End(errMsg)

		log.Printf(
			"[TRACE] Graph after step %T:\n\n%s",
			step, g.StringWithNodeTypes())

		if err != nil {
			return g, err
		}
	}

	// Validate the graph structure
	if b.Validate {
		if err := g.Validate(); err != nil {
			log.Printf("[ERROR] Graph validation failed. Graph:\n\n%s", g.String())
			return nil, err
		}
	}

	return g, nil
}
