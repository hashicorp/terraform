package command

import (
	"fmt"
	"strings"

	"github.com/mitchellh/go-libucl"
)

// FlagVar is a flag.Value implementation for parsing user variables
// from the command-line in the format of '-var key=value'.
type FlagVar map[string]string

func (v *FlagVar) String() string {
	return ""
}

func (v *FlagVar) Set(raw string) error {
	idx := strings.Index(raw, "=")
	if idx == -1 {
		return fmt.Errorf("No '=' value in arg: %s", raw)
	}

	if *v == nil {
		*v = make(map[string]string)
	}

	key, value := raw[0:idx], raw[idx+1:]
	(*v)[key] = value
	return nil
}

// FlagVarFile is a flag.Value implementation for parsing user variables
// from the command line in the form of files. i.e. '-var-file=foo'
type FlagVarFile map[string]string

func (v *FlagVarFile) String() string {
	return ""
}

func (v *FlagVarFile) Set(raw string) error {
	vs, err := loadVarFile(raw)
	if err != nil {
		return err
	}

	if *v == nil {
		*v = make(map[string]string)
	}

	for key, value := range vs {
		(*v)[key] = value
	}

	return nil
}

const libuclParseFlags = libucl.ParserNoTime

func loadVarFile(path string) (map[string]string, error) {
	var obj *libucl.Object

	parser := libucl.NewParser(libuclParseFlags)
	err := parser.AddFile(path)
	if err == nil {
		obj = parser.Object()
		defer obj.Close()
	}
	defer parser.Close()

	if err != nil {
		return nil, err
	}

	var result map[string]string
	if err := obj.Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}
