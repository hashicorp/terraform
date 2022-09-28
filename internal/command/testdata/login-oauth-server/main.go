//go:build ignore
// +build ignore

// This file is a helper for those doing _manual_ testing of "terraform login"
// and/or "terraform logout" and want to start up a test OAuth server in a
// separate process for convenience:
//
//    go run ./command/testdata/login-oauth-server/main.go :8080
//
// This is _not_ the main way to use this oauthserver package. For automated
// test code, import it as a normal Go package instead:
//
//     import oauthserver "github.com/hashicorp/terraform/internal/command/testdata/login-oauth-server"

package main

import (
	"fmt"
	"net"
	"net/http"
	"os"

	oauthserver "github.com/hashicorp/terraform/internal/command/testdata/login-oauth-server"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go run ./command/testdata/login-oauth-server/main.go <listen-address>")
		os.Exit(1)
	}

	host, port, err := net.SplitHostPort(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Invalid address: %s", err)
		os.Exit(1)
	}

	if host == "" {
		host = "127.0.0.1"
	}
	addr := fmt.Sprintf("%s:%s", host, port)

	fmt.Printf("Will listen on %s...\n", addr)
	fmt.Printf(
		configExampleFmt,
		fmt.Sprintf("http://%s:%s/authz", host, port),
		fmt.Sprintf("http://%s:%s/token", host, port),
		fmt.Sprintf("http://%s:%s/revoke", host, port),
	)

	server := &http.Server{
		Addr:    addr,
		Handler: oauthserver.Handler,
	}
	err = server.ListenAndServe()
	fmt.Fprintln(os.Stderr, err.Error())
}

const configExampleFmt = `
host "login-test.example.com" {
  services = {
    "login.v1" = {
      authz       = %q
      token       = %q
      client      = "placeholder"
      grant_types = ["code", "password"]
    }
    "logout.v1" = %q
  }
}

`
