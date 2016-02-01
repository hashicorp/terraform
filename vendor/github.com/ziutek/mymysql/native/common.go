package native

import (
	"io"
	"runtime"
)

var tab8s = "        "

func catchError(err *error) {
	if pv := recover(); pv != nil {
		switch e := pv.(type) {
		case runtime.Error:
			panic(pv)
		case error:
			if e == io.EOF {
				*err = io.ErrUnexpectedEOF
			} else {
				*err = e
			}
		default:
			panic(pv)
		}
	}
}
