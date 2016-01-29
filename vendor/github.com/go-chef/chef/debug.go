// +build debug

package chef

import "log"

func debug(fmt string, args ...interface{}) {
	log.Printf(fmt, args...)
}
