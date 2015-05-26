package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mitchellh/panicwrap"
)

// This output is shown if a panic happens.
const panicOutput = `

!!!!!!!!!!!!!!!!!!!!!!!!!!! TERRAFORM CRASH !!!!!!!!!!!!!!!!!!!!!!!!!!!!

Terraform crashed! This is always indicative of a bug within Terraform.
A crash log has been placed at "crash.log" relative to your current
working directory. It would be immensely helpful if you could please
report the crash with Terraform[1] so that we can fix this.

When reporting bugs, please include your terraform version. That
information is available on the first line of crash.log. You can also
get it by running 'terraform --version' on the command line.

[1]: https://github.com/hashicorp/terraform/issues

!!!!!!!!!!!!!!!!!!!!!!!!!!! TERRAFORM CRASH !!!!!!!!!!!!!!!!!!!!!!!!!!!!
`

// panicHandler is what is called by panicwrap when a panic is encountered
// within Terraform. It is guaranteed to run after the resulting process has
// exited so we can take the log file, add in the panic, and store it
// somewhere locally.
func panicHandler(logF *os.File) panicwrap.HandlerFunc {
	return func(m string) {
		// Right away just output this thing on stderr so that it gets
		// shown in case anything below fails.
		fmt.Fprintf(os.Stderr, fmt.Sprintf("%s\n", m))

		// Create the crash log file where we'll write the logs
		f, err := os.Create("crash.log")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create crash log file: %s", err)
			return
		}
		defer f.Close()

		// Seek the log file back to the beginning
		if _, err = logF.Seek(0, 0); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to seek log file for crash: %s", err)
			return
		}

		// Copy the contents to the crash file. This will include
		// the panic that just happened.
		if _, err = io.Copy(f, logF); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write crash log: %s", err)
			return
		}

		// Tell the user a crash occurred in some helpful way that
		// they'll hopefully notice.
		fmt.Printf("\n\n")
		fmt.Println(strings.TrimSpace(panicOutput))
	}
}
