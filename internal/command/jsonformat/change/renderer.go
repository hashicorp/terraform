package change

// Renderer renders a specific change.
//
// Implementations should handle unique functionality relevant to the specific
// change type. Any common functionality shared between multiple change
// renderers should be pushed into the Change structure itself.
type Renderer interface {
	Render(change Change, indent int, opts RenderOpts) string
	Warnings(change Change, indent int) []string
}

// NoWarningsRenderer defines a Warnings function that returns an empty list of
// warnings. This can be used by other renderers to ensure we don't see lots of
// repeats of this empty function.
type NoWarningsRenderer struct{}

// Warnings returns an empty slice, as the name NoWarningsRenderer suggests.
func (render NoWarningsRenderer) Warnings(change Change, indent int) []string {
	return nil
}

// RenderOpts contains options that can control how the Renderer.Render function
// will render.
//
// For now, we haven't implemented any of the Renderer functionality, so we have
// no options currently.
type RenderOpts struct{}

// Clone returns a new RenderOpts object, that matches the original but can be
// edited without changing the original.
func (opts RenderOpts) Clone() RenderOpts {
	return RenderOpts{}
}
