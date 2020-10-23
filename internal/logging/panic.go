package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/mitchellh/panicwrap"
)

// This output is shown if a panic happens.
const panicOutput = `

!!!!!!!!!!!!!!!!!!!!!!!!!!! TERRAFORM CRASH !!!!!!!!!!!!!!!!!!!!!!!!!!!!

Terraform crashed! This is always indicative of a bug within Terraform.
A crash log has been placed at %[1]q relative to your current
working directory. It would be immensely helpful if you could please
report the crash with Terraform[1] so that we can fix this.

When reporting bugs, please include your terraform version. That
information is available on the first line of crash.log. You can also
get it by running 'terraform --version' on the command line.

SECURITY WARNING: the %[1]q file that was created may contain 
sensitive information that must be redacted before it is safe to share 
on the issue tracker.

[1]: https://github.com/hashicorp/terraform/issues

!!!!!!!!!!!!!!!!!!!!!!!!!!! TERRAFORM CRASH !!!!!!!!!!!!!!!!!!!!!!!!!!!!
`

// panicHandler is what is called by panicwrap when a panic is encountered
// within Terraform. It is guaranteed to run after the resulting process has
// exited so we can take the log file, add in the panic, and store it
// somewhere locally.
func PanicHandler(tmpLogPath string) panicwrap.HandlerFunc {
	return func(m string) {
		// Create the crash log file where we'll write the logs
		f, err := ioutil.TempFile(".", "crash.*.log")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create crash log file: %s", err)
			return
		}
		defer f.Close()

		tmpLog, err := os.Open(tmpLogPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open log file %q: %v\n", tmpLogPath, err)
			return
		}
		defer tmpLog.Close()

		// Copy the contents to the crash file. This will include
		// the panic that just happened.
		if _, err = io.Copy(f, tmpLog); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write crash log: %s", err)
			return
		}

		// add the trace back to the log
		f.WriteString("\n" + m)

		// Tell the user a crash occurred in some helpful way that
		// they'll hopefully notice.
		fmt.Printf("\n\n")
		fmt.Printf(panicOutput, f.Name())
	}
}
