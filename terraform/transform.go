package terraform

// GraphTransformer is the interface that transformers implement. This
// interface is only for transforms that need entire graph visibility.
type GraphTransformer interface {
	Transform(*Graph) error
}
