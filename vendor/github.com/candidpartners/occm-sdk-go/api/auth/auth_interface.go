// Package implements OCCM Auth API
package auth

type AuthAPIProto interface {
	Login(string, string) error
  Logout() error
}

var _ AuthAPIProto = (*AuthAPI)(nil)
