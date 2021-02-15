package logging

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	"github.com/hashicorp/go-hclog"
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

const pluginPanicOutput = `
Stack trace from the %[1]s plugin:

%s

Error: The %[1]s plugin crashed!

This is always indicative of a bug within the plugin. It would be immensely
helpful if you could report the crash with the plugin's maintainers so that it
can be fixed. The output above should help diagnose the issue.
`

// PluginPanics returns a series of provider panics that were collected during
// execution, and formatted for output.
func PluginPanics() []string {
	return panics.allPanics()
}

// panicRecorder provides a registry to check for plugin panics that may have
// happened when a plugin suddenly terminates.
type panicRecorder struct {
	sync.Mutex

	// panics maps the plugin name to the panic output lines received from
	// the logger.
	panics map[string][]string

	// maxLines is the max number of lines we'll record after seeing a
	// panic header. Since this is going to be printed in the UI output, we
	// don't want to destroy the scrollback. In most cases, the first few lines
	// of the stack trace is all that are required.
	maxLines int
}

// registerPlugin returns an accumulator function which will accept lines of
// a panic stack trace to collect into an error when requested.
func (p *panicRecorder) registerPlugin(name string) func(string) {
	p.Lock()
	defer p.Unlock()

	// In most cases we shouldn't be starting a plugin if it already
	// panicked, but clear out previous entries just in case.
	delete(p.panics, name)

	count := 0

	// this callback is used by the logger to store panic output
	return func(line string) {
		p.Lock()
		defer p.Unlock()

		// stop recording if there are too many lines.
		if count > p.maxLines {
			return
		}
		count++

		p.panics[name] = append(p.panics[name], line)
	}
}

func (p *panicRecorder) allPanics() []string {
	p.Lock()
	defer p.Unlock()

	var res []string
	for name, lines := range p.panics {
		if len(lines) == 0 {
			continue
		}

		res = append(res, fmt.Sprintf(pluginPanicOutput, name, strings.Join(lines, "\n")))
	}
	return res
}

// logPanicWrapper wraps an hclog.Logger and intercepts and records any output
// that appears to be a panic.
type logPanicWrapper struct {
	hclog.Logger
	panicRecorder func(string)
	inPanic       bool
}

// go-plugin will create a new named logger for each plugin binary.
func (l *logPanicWrapper) Named(name string) hclog.Logger {
	return &logPanicWrapper{
		Logger:        l.Logger.Named(name),
		panicRecorder: panics.registerPlugin(name),
	}
}

// we only need to implement Debug, since that is the default output level used
// by go-plugin when encountering unstructured output on stderr.
func (l *logPanicWrapper) Debug(msg string, args ...interface{}) {
	// We don't have access to the binary itself, so guess based on the stderr
	// output if this is the start of the traceback. An occasional false
	// positive shouldn't be a big deal, since this is only retrieved after an
	// error of some sort.

	panicPrefix := strings.HasPrefix(msg, "panic: ") || strings.HasPrefix(msg, "fatal error: ")

	l.inPanic = l.inPanic || panicPrefix

	if l.inPanic && l.panicRecorder != nil {
		l.panicRecorder(msg)
	}

	// If we have logging turned on, we need to prevent panicwrap from seeing
	// this as a core panic. This can be done by obfuscating the panic error
	// line.
	if panicPrefix {
		colon := strings.Index(msg, ":")
		msg = strings.ToUpper(msg[:colon]) + msg[colon:]
	}

	l.Logger.Debug(msg, args...)
}
