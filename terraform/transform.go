package terraform

import (
	"github.com/hashicorp/terraform/dag"
)

// GraphTransformer is the interface that transformers implement. This
// interface is only for transforms that need entire graph visibility.
type GraphTransformer interface {
	Transform(*dag.Graph) error
}
