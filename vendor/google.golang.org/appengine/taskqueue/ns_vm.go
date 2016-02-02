// +build !appengine

package taskqueue

import (
	"golang.org/x/net/context"

	"google.golang.org/appengine/internal"
)

func getDefaultNamespace(ctx context.Context) string {
	return internal.IncomingHeaders(ctx).Get(defaultNamespace)
}
