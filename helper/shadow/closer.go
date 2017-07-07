package shadow

import (
	"fmt"
	"io"
	"reflect"

	"github.com/hashicorp/go-multierror"
	"github.com/mitchellh/reflectwalk"
)

// Close will close all shadow values within the given structure.
//
// This uses reflection to walk the structure, find all shadow elements,
// and close them. Currently this will only find struct fields that are
// shadow values, and not slice elements, etc.
func Close(v interface{}) error {
	// We require a pointer so we can address the internal fields
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("value must be a pointer")
	}

	// Walk and close
	var w closeWalker
	if err := reflectwalk.Walk(v, &w); err != nil {
		return err
	}

	return w.Err
}

type closeWalker struct {
	Err error
}

func (w *closeWalker) Struct(reflect.Value) error {
	// Do nothing. We implement this for reflectwalk.StructWalker
	return nil
}

var closerType = reflect.TypeOf((*io.Closer)(nil)).Elem()

func (w *closeWalker) StructField(f reflect.StructField, v reflect.Value) error {
	// Not sure why this would be but lets avoid some panics
	if !v.IsValid() {
		return nil
	}

	// Empty for exported, so don't check unexported fields
	if f.PkgPath != "" {
		return nil
	}

	// Verify the io.Closer is in this package
	typ := v.Type()
	if typ.PkgPath() != "github.com/hashicorp/terraform/helper/shadow" {
		return nil
	}

	var closer io.Closer
	if v.Type().Implements(closerType) {
		closer = v.Interface().(io.Closer)
	} else if v.CanAddr() {
		// The Close method may require a pointer receiver, but we only have a value.
		v := v.Addr()
		if v.Type().Implements(closerType) {
			closer = v.Interface().(io.Closer)
		}
	}

	if closer == nil {
		return reflectwalk.SkipEntry
	}

	// Close it
	if err := closer.Close(); err != nil {
		w.Err = multierror.Append(w.Err, err)
	}

	// Don't go into the struct field
	return reflectwalk.SkipEntry
}
