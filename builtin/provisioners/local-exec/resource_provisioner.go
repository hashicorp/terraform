package localexec

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/armon/circbuf"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
)

const (
	// maxBufSize limits how much output we collect from a local
	// invocation. This is to prevent TF memory usage from growing
	// to an enormous amount due to a faulty process.
	maxBufSize = 8 * 1024
)

func Provisioner() terraform.ResourceProvisioner {
	return &schema.Provisioner{
		Schema: map[string]*schema.Schema{
			"command": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"interpreter": &schema.Schema{
				Type:     schema.TypeList,
				Elem:     &schema.Schema{Type: schema.TypeString},
				Optional: true,
			},
			"working_dir": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"environment": &schema.Schema{
				Type:     schema.TypeMap,
				Optional: true,
			},
		},

		ApplyFunc: applyFn,
	}
}

func applyFn(ctx context.Context) error {
	data := ctx.Value(schema.ProvConfigDataKey).(*schema.ResourceData)
	o := ctx.Value(schema.ProvOutputKey).(terraform.UIOutput)

	command := data.Get("command").(string)
	if command == "" {
		return fmt.Errorf("local-exec provisioner command must be a non-empty string")
	}

	// Execute the command with env
	environment := data.Get("environment").(map[string]interface{})

	var env []string
	env = make([]string, len(environment))
	for k := range environment {
		entry := fmt.Sprintf("%s=%s", k, environment[k].(string))
		env = append(env, entry)
	}

	// Execute the command using a shell
	interpreter := data.Get("interpreter").([]interface{})

	var cmdargs []string
	if len(interpreter) > 0 {
		for _, i := range interpreter {
			if arg, ok := i.(string); ok {
				cmdargs = append(cmdargs, arg)
			}
		}
	} else {
		if runtime.GOOS == "windows" {
			cmdargs = []string{"cmd", "/C"}
		} else {
			cmdargs = []string{"/bin/sh", "-c"}
		}
	}
	cmdargs = append(cmdargs, command)

	workingdir := data.Get("working_dir").(string)

	// Setup the reader that will read the output from the command.
	// We use an os.Pipe so that the *os.File can be passed directly to the
	// process, and not rely on goroutines copying the data which may block.
	// See golang.org/issue/18874
	pr, pw, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to initialize pipe for output: %s", err)
	}

	var cmdEnv []string
	cmdEnv = os.Environ()
	cmdEnv = append(cmdEnv, env...)

	// Setup the command
	cmd := exec.CommandContext(ctx, cmdargs[0], cmdargs[1:]...)
	cmd.Stderr = pw
	cmd.Stdout = pw
	// Dir specifies the working directory of the command.
	// If Dir is the empty string (this is default), runs the command
	// in the calling process's current directory.
	cmd.Dir = workingdir
	// Env specifies the environment of the command.
	// By default will use the calling process's environment
	cmd.Env = cmdEnv

	output, _ := circbuf.NewBuffer(maxBufSize)

	// Write everything we read from the pipe to the output buffer too
	tee := io.TeeReader(pr, output)

	// copy the teed output to the UI output
	copyDoneCh := make(chan struct{})
	go copyOutput(o, tee, copyDoneCh)

	// Output what we're about to run
	o.Output(fmt.Sprintf("Executing: %q", cmdargs))

	// Start the command
	err = cmd.Start()
	if err == nil {
		err = cmd.Wait()
	}

	// Close the write-end of the pipe so that the goroutine mirroring output
	// ends properly.
	pw.Close()

	// Cancelling the command may block the pipe reader if the file descriptor
	// was passed to a child process which hasn't closed it. In this case the
	// copyOutput goroutine will just hang out until exit.
	select {
	case <-copyDoneCh:
	case <-ctx.Done():
	}

	if err != nil {
		return fmt.Errorf("Error running command '%s': %v. Output: %s",
			command, err, output.Bytes())
	}

	return nil
}

func copyOutput(o terraform.UIOutput, r io.Reader, doneCh chan<- struct{}) {
	defer close(doneCh)
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}
