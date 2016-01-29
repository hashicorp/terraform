# panicwrap

panicwrap is a Go library that re-executes a Go binary and monitors stderr
output from the binary for a panic. When it finds a panic, it executes a
user-defined handler function. Stdout, stderr, stdin, signals, and exit
codes continue to work as normal, making the existence of panicwrap mostly
invisible to the end user until a panic actually occurs.

Since a panic is truly a bug in the program meant to crash the runtime,
globally catching panics within Go applications is not supposed to be possible.
Despite this, it is often useful to have a way to know when panics occur.
panicwrap allows you to do something with these panics, such as writing them
to a file, so that you can track when panics occur.

panicwrap is ***not a panic recovery system***. Panics indicate serious
problems with your application and _should_ crash the runtime. panicwrap
is just meant as a way to monitor for panics. If you still think this is
the worst idea ever, read the section below on why.

## Features

* **SIMPLE!**
* Works with all Go applications on all platforms Go supports
* Custom behavior when a panic occurs
* Stdout, stderr, stdin, exit codes, and signals continue to work as
  expected.

## Usage

Using panicwrap is simple. It behaves a lot like `fork`, if you know
how that works. A basic example is shown below.

Because it would be sad to panic while capturing a panic, it is recommended
that the handler functions for panicwrap remain relatively simple and well
tested. panicwrap itself contains many tests.

```go
package main

import (
	"fmt"
	"github.com/mitchellh/panicwrap"
	"os"
)

func main() {
	exitStatus, err := panicwrap.BasicWrap(panicHandler)
	if err != nil {
		// Something went wrong setting up the panic wrapper. Unlikely,
		// but possible.
		panic(err)
	}

	// If exitStatus >= 0, then we're the parent process and the panicwrap
	// re-executed ourselves and completed. Just exit with the proper status.
	if exitStatus >= 0 {
		os.Exit(exitStatus)
	}

	// Otherwise, exitStatus < 0 means we're the child. Continue executing as
	// normal...

	// Let's say we panic
	panic("oh shucks")
}

func panicHandler(output string) {
	// output contains the full output (including stack traces) of the
	// panic. Put it in a file or something.
	fmt.Printf("The child panicked:\n\n%s\n", output)
	os.Exit(1)
}
```

## How Does it Work?

panicwrap works by re-executing the running program (retaining arguments,
environmental variables, etc.) and monitoring the stderr of the program.
Since Go always outputs panics in a predictable way with a predictable
exit code, panicwrap is able to reliably detect panics and allow the parent
process to handle them.

## WHY?! Panics should CRASH!

Yes, panics _should_ crash. They are 100% always indicative of bugs and having
information on a production server or application as to what caused the panic is critical.

### User Facing

In user-facing programs (programs like
[Packer](http://github.com/mitchellh/packer) or
[Docker](http://github.com/dotcloud/docker)), it is up to the user to
report such panics. This is unreliable, at best, and it would be better if the
program could have a way to automatically report panics. panicwrap provides
a way to do this.

### Server

For backend applications, it is easier to detect crashes (since the application exits) 
and having an idea as to why the crash occurs is equally important; 
particularly on a production server. 

At [HashiCorp](http://www.hashicorp.com),
we use panicwrap to log panics to timestamped files with some additional
data (configuration settings at the time, environmental variables, etc.)

The goal of panicwrap is _not_ to hide panics. It is instead to provide
a clean mechanism for capturing them and ultimately crashing.
