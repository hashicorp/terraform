package terraform

import (
	"sync"

	"github.com/hashicorp/terraform/internal/configs"
	"github.com/hashicorp/terraform/internal/dag"
)

type BoundaryTransformer struct {
	Config    *configs.Config
	CloseNode *NodeBoundaryCloser
}

func NewBoundaryTransformer(c *configs.Config) *BoundaryTransformer {
	return &BoundaryTransformer{
		Config: c,
		CloseNode: &NodeBoundaryCloser{
			lock: &sync.Mutex{},
		},
	}
}

func (b *BoundaryTransformer) Proxies() *BoundaryTransformerProxy {
	return &BoundaryTransformerProxy{
		Config:    b.Config,
		CloseNode: b.CloseNode,
	}
}

func (b *BoundaryTransformer) Closer() *BoundaryTransformerCloser {
	return &BoundaryTransformerCloser{
		Config:    b.Config,
		CloseNode: b.CloseNode,
	}
}

type BoundaryTransformerProxy struct {
	Config    *configs.Config
	CloseNode *NodeBoundaryCloser
}

func (b *BoundaryTransformerProxy) Transform(g *Graph) error {
	if b.Config == nil || len(b.Config.Module.Boundary) == 0 {
		return nil
	}

	for name, conn := range b.Config.Module.Boundary {
		v := &NodeBoundary{
			ConnectionName: name,
			Config:         conn.Config,
			Connection:     conn.Connection,
			Schema:         conn.Schema,
			DeclRange:      &conn.DeclRange,
			CloseNode:      b.CloseNode,
		}
		g.Add(v)
	}

	return nil
}

type BoundaryTransformerCloser struct {
	Config    *configs.Config
	CloseNode *NodeBoundaryCloser
}

func (b *BoundaryTransformerCloser) Transform(g *Graph) error {
	if b.Config == nil || len(b.Config.Module.Boundary) == 0 {
		return nil
	}

	// Make the closing node depend on everything so we don't close the proxies
	// too early
	g.Add(b.CloseNode)

	for _, v := range g.Vertices() {
		if v == b.CloseNode {
			continue
		}

		if g.UpEdges(v).Len() == 0 {
			g.Connect(dag.BasicEdge(b.CloseNode, v))
		}
	}

	return nil
}
