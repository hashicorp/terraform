package terminal

import (
	"fmt"
	"io"
	"io/ioutil"
)

type TeePrinter struct {
	disableTerminalOutput bool
	outputBucket          io.Writer
	stdout                io.Writer
}

func NewTeePrinter(w io.Writer) *TeePrinter {
	return &TeePrinter{
		outputBucket: ioutil.Discard,
		stdout:       w,
	}
}

func (t *TeePrinter) SetOutputBucket(bucket io.Writer) {
	if bucket == nil {
		bucket = ioutil.Discard
	}

	t.outputBucket = bucket
}

func (t *TeePrinter) Print(values ...interface{}) (int, error) {
	str := fmt.Sprint(values...)
	t.saveOutputToBucket(str)
	if !t.disableTerminalOutput {
		return fmt.Fprint(t.stdout, str)
	}
	return 0, nil
}

func (t *TeePrinter) Printf(format string, a ...interface{}) (int, error) {
	str := fmt.Sprintf(format, a...)
	t.saveOutputToBucket(str)
	if !t.disableTerminalOutput {
		return fmt.Fprint(t.stdout, str)
	}
	return 0, nil
}

func (t *TeePrinter) Println(values ...interface{}) (int, error) {
	str := fmt.Sprint(values...)
	t.saveOutputToBucket(str)
	if !t.disableTerminalOutput {
		return fmt.Fprintln(t.stdout, str)
	}
	return 0, nil
}

func (t *TeePrinter) DisableTerminalOutput(disable bool) {
	t.disableTerminalOutput = disable
}

func (t *TeePrinter) saveOutputToBucket(output string) {
	_, _ = t.outputBucket.Write([]byte(Decolorize(output)))
}
