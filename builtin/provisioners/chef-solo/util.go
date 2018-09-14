package chefsolo

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"text/template"

	"github.com/hashicorp/terraform/communicator"
	"github.com/hashicorp/terraform/communicator/remote"
	"github.com/hashicorp/terraform/terraform"
	"github.com/mitchellh/go-linereader"
)

func getCommunicator(ctx context.Context, o terraform.UIOutput, s *terraform.InstanceState) (communicator.Communicator, error) {
	comm, err := communicator.New(s)
	if err != nil {
		return nil, err
	}
	retryCtx, cancel := context.WithTimeout(ctx, comm.Timeout())
	defer cancel()

	// Wait and retry until we establish the connection
	err = communicator.Retry(retryCtx, func() error {
		return comm.Connect(o)
	})
	if err != nil {
		return nil, err
	}
	defer comm.Disconnect()

	return comm, err
}

func getGuestOSType(state *terraform.InstanceState) (string, error) {
	switch t := state.Ephemeral.ConnInfo["type"]; t {
	case "ssh", "": // The default connection type is ssh, so if the type is empty assume ssh
		return "unix", nil
	case "winrm":
		return "windows", nil
	default:
		return "", fmt.Errorf("Can't find OS type based on the connection type: %s", t)
	}
}

func setIfEmpty(variable interface{}, defaultVal interface{}) {
	switch v := variable.(type) {
	case *string:
		if *v == "" {
			*v = defaultVal.(string)
		}
	case *bool:
		if *v == false {
			*v = defaultVal.(bool)
		}
	case *int:
		if *v == 0 {
			*v = defaultVal.(int)
		}
	default:
		fmt.Errorf("Unsupported type %T", v)
	}
}

// parses text as a template and executes it using the data
func renderTemplate(text string, data interface{}) string {
	var buff bytes.Buffer
	template.Must(template.New("").Parse(text)).Execute(&buff, data)
	return buff.String()
}

func getStringList(v interface{}) []string {
	var result []string

	switch v := v.(type) {
	case nil:
		return result
	case []interface{}:
		for _, vv := range v {
			if vv, ok := vv.(string); ok {
				result = append(result, vv)
			}
		}
		return result
	default:
		panic(fmt.Sprintf("Unsupported type: %T", v))
	}
}

func (p *provisioner) createDir(o terraform.UIOutput, comm communicator.Communicator, dir string) error {
	cmd := fmt.Sprintf(osDefaults[p.GuestOSType].createDirCommand, dir, dir)

	o.Output(fmt.Sprintf("Creating directory: %s", dir))
	if err := p.runCommand(o, comm, cmd); err != nil {
		return err
	}
	return nil
}

func (p *provisioner) uploadDir(o terraform.UIOutput, comm communicator.Communicator, dst string, src string) error {
	if src == "" {
		return nil
	}

	if err := p.createDir(o, comm, dst); err != nil {
		return err
	}

	src = strings.TrimSuffix(src, "/") + "/"

	o.Output(fmt.Sprintf("Uploading local directory %s to %s", src, dst))
	return comm.UploadDir(dst, src)
}

func (p *provisioner) uploadFile(o terraform.UIOutput, comm communicator.Communicator, dst string, src string) error {
	if src == "" {
		return nil
	}
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	return comm.Upload(dst, f)
}

// runCommand is used to run already prepared commands
func (p *provisioner) runCommand(o terraform.UIOutput, comm communicator.Communicator, command string) error {
	// Unless prevented, prefix the command with sudo
	if !p.PreventSudo && p.GuestOSType == "unix" {
		command = "sudo " + command
	}

	outR, outW := io.Pipe()
	errR, errW := io.Pipe()
	go p.copyOutput(o, outR)
	go p.copyOutput(o, errR)
	defer outW.Close()
	defer errW.Close()

	cmd := &remote.Cmd{
		Command: command,
		Stdout:  outW,
		Stderr:  errW,
	}

	err := comm.Start(cmd)
	if err != nil {
		return fmt.Errorf("Error executing command %q: %v", cmd.Command, err)
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func (p *provisioner) copyOutput(o terraform.UIOutput, r io.Reader) {
	lr := linereader.New(r)
	for line := range lr.Ch {
		o.Output(line)
	}
}

// Output implementation of terraform.UIOutput interface
func (p *provisioner) Output(output string) {
	logFile := path.Join("logfiles", "dummy-node")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		log.Printf("Error creating logfile %s: %v", logFile, err)
		return
	}
	defer f.Close()

	// These steps are needed to remove any ANSI escape codes used to colorize
	// the output and to make sure we have proper line endings before writing
	// the string to the logfile.
	re := regexp.MustCompile(`\x1b\[[0-9;]+m`)
	output = re.ReplaceAllString(output, "")
	output = strings.Replace(output, "\r", "\n", -1)
}
