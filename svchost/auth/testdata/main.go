package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// This is a simple program that implements the "helper program" protocol
// for the svchost/auth package for unit testing purposes.

func main() {
	args := os.Args

	if len(args) < 3 {
		die("not enough arguments\n")
	}

	host := args[2]
	switch args[1] {
	case "get":
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
	case "store":
		dataSrc, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			die("invalid input: %s", err)
		}
		var data map[string]interface{}
		err = json.Unmarshal(dataSrc, &data)

		switch host {
		case "example.com":
			if data["token"] != "example-token" {
				die("incorrect token value to store")
			}
		default:
			die("can't store credentials for %s", host)
		}
	case "forget":
		switch host {
		case "example.com":
			// okay!
		default:
			die("can't forget credentials for %s", host)
		}
	default:
		die("unknown subcommand %q\n", args[1])
	}
}

func die(f string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, fmt.Sprintf(f, args...))
	os.Exit(1)
}
