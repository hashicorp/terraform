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
type RenderOpts struct {

	// overrideNullSuffix tells the Renderer not to display the `-> null` suffix
	// that is normally displayed when an element, attribute, or block is
	// deleted.
	//
	// The presence of this suffix is decided by the parent changes of a given
	// change, as such we provide this as an option instead of trying to
	// calculate it inside a specific renderer.
	overrideNullSuffix bool

	// showUnchangedChildren instructs the Renderer to render all children of a
	// given complex change, instead of hiding unchanged items and compressing
	// them into a single line.
	//
	// This is generally decided by the parent change (mainly lists) and so is
	// passed in as a private option.
	showUnchangedChildren bool
}

// Clone returns a new RenderOpts object, that matches the original but can be
// edited without changing the original.
func (opts RenderOpts) Clone() RenderOpts {
	return RenderOpts{
		overrideNullSuffix: opts.overrideNullSuffix,
	}
}
