package change

import "fmt"

func TypeChange(before, after Change) Renderer {
	return &typeChangeRenderer{
		before: before,
		after:  after,
	}
}

type typeChangeRenderer struct {
	NoWarningsRenderer

	before Change
	after  Change
}

func (renderer typeChangeRenderer) Render(change Change, indent int, opts RenderOpts) string {
	opts.overrideNullSuffix = true // Never render null suffix for children of type changes.
	return fmt.Sprintf("%s [yellow]->[reset] %s", renderer.before.Render(indent, opts), renderer.after.Render(indent, opts))
}
