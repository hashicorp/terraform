package cloudplugin

import (
	"io"
)

type Cloud1 interface {
	Execute(args []string, stdout, stderr io.Writer) int
}
