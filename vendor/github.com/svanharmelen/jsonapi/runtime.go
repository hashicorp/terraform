package jsonapi

import (
	"crypto/rand"
	"fmt"
	"io"
	"reflect"
	"time"
)

type Event int

const (
	UnmarshalStart Event = iota
	UnmarshalStop
	MarshalStart
	MarshalStop
)

type Runtime struct {
	ctx map[string]interface{}
}

type Events func(*Runtime, Event, string, time.Duration)

var Instrumentation Events

func NewRuntime() *Runtime { return &Runtime{make(map[string]interface{})} }

func (r *Runtime) WithValue(key string, value interface{}) *Runtime {
	r.ctx[key] = value

	return r
}

func (r *Runtime) Value(key string) interface{} {
	return r.ctx[key]
}

func (r *Runtime) Instrument(key string) *Runtime {
	return r.WithValue("instrument", key)
}

func (r *Runtime) shouldInstrument() bool {
	return Instrumentation != nil
}

func (r *Runtime) UnmarshalPayload(reader io.Reader, model interface{}) error {
	return r.instrumentCall(UnmarshalStart, UnmarshalStop, func() error {
		return UnmarshalPayload(reader, model)
	})
}

func (r *Runtime) UnmarshalManyPayload(reader io.Reader, kind reflect.Type) (elems []interface{}, err error) {
	r.instrumentCall(UnmarshalStart, UnmarshalStop, func() error {
		elems, err = UnmarshalManyPayload(reader, kind)
		return err
	})

	return
}

func (r *Runtime) MarshalPayload(w io.Writer, model interface{}) error {
	return r.instrumentCall(MarshalStart, MarshalStop, func() error {
		return MarshalPayload(w, model)
	})
}

func (r *Runtime) instrumentCall(start Event, stop Event, c func() error) error {
	if !r.shouldInstrument() {
		return c()
	}

	instrumentationGUID, err := newUUID()
	if err != nil {
		return err
	}

	begin := time.Now()
	Instrumentation(r, start, instrumentationGUID, time.Duration(0))

	if err := c(); err != nil {
		return err
	}

	diff := time.Duration(time.Now().UnixNano() - begin.UnixNano())
	Instrumentation(r, stop, instrumentationGUID, diff)

	return nil
}

// citation: http://play.golang.org/p/4FkNSiUDMg
func newUUID() (string, error) {
	uuid := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, uuid); err != nil {
		return "", err
	}
	// variant bits; see section 4.1.1
	uuid[8] = uuid[8]&^0xc0 | 0x80
	// version 4 (pseudo-random); see section 4.1.3
	uuid[6] = uuid[6]&^0xf0 | 0x40
	return fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
