package formatters

import (
	"fmt"
	"reflect"
)

func MapStr(args interface{}) []string {
	r := reflect.ValueOf(args)
	rval := make([]string, r.Len())
	for i := 0; i < r.Len(); i++ {
		rval[i] = r.Index(i).Interface().(fmt.Stringer).String()
	}
	return rval

}
