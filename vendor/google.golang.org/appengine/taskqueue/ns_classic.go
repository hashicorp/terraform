// +build appengine

package taskqueue

import (
	basepb "appengine_internal/base"

	"golang.org/x/net/context"

	"google.golang.org/appengine/internal"
)

func getDefaultNamespace(ctx context.Context) string {
	c := internal.ClassicContextFromContext(ctx)
	s := &basepb.StringProto{}
	c.Call("__go__", "GetDefaultNamespace", &basepb.VoidProto{}, s, nil)
	return s.GetValue()
}
