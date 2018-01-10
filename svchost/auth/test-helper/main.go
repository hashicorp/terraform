package main

import (
	"fmt"
	"os"
)

// This is a simple program that implements the "helper program" protocol
// for the svchost/auth package for unit testing purposes.

func main() {
	args := os.Args

	if len(args) < 3 {
		die("not enough arguments\n")
	}

	if args[1] != "get" {
		die("unknown subcommand %q\n", args[1])
	}

	host := args[2]

	switch host {
	case "example.com":
		fmt.Print(`{"token":"example-token"}`)
	case "other-cred-type.example.com":
		fmt.Print(`{"username":"alfred"}`) // unrecognized by main program
	case "fail.example.com":
		die("failing because you told me to fail\n")
	default:
		fmt.Print("{}") // no credentials available
	}
}

func die(f string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf(f, args...))
	os.Exit(1)
}
