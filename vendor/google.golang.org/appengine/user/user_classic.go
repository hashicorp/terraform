// +build appengine

package user

import (
	"appengine/user"

	"golang.org/x/net/context"

	"google.golang.org/appengine/internal"
)

func Current(ctx context.Context) *User {
	u := user.Current(internal.ClassicContextFromContext(ctx))
	if u == nil {
		return nil
	}
	// Map appengine/user.User to this package's User type.
	return &User{
		Email:             u.Email,
		AuthDomain:        u.AuthDomain,
		Admin:             u.Admin,
		ID:                u.ID,
		FederatedIdentity: u.FederatedIdentity,
		FederatedProvider: u.FederatedProvider,
	}
}

func IsAdmin(ctx context.Context) bool {
	return user.IsAdmin(internal.ClassicContextFromContext(ctx))
}
