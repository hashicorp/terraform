package terraform

import (
	"bytes"
	"encoding/gob"
	"errors"
	"io"
	"reflect"
	"sync"
	"testing"

	"github.com/mitchellh/hashstructure"
)

func TestReadWriteStateV1(t *testing.T) {
	state := &StateV1{
		Resources: map[string]*ResourceStateV1{
			"foo": &ResourceStateV1{
				ID: "bar",
				ConnInfo: map[string]string{
					"type":     "ssh",
					"user":     "root",
					"password": "supersecret",
				},
			},
		},
	}

	// Checksum before the write
	chksum, err := hashstructure.Hash(state, nil)
	if err != nil {
		t.Fatalf("hash: %s", err)
	}

	buf := new(bytes.Buffer)
	if err := testWriteStateV1(state, buf); err != nil {
		t.Fatalf("err: %s", err)
	}

	// Checksum after the write
	chksumAfter, err := hashstructure.Hash(state, nil)
	if err != nil {
		t.Fatalf("hash: %s", err)
	}

	if chksumAfter != chksum {
		t.Fatalf("structure changed during serialization!")
	}

	actual, err := ReadStateV1(buf)
	if err != nil {
		t.Fatalf("err: %s", err)
	}

	// ReadState should not restore sensitive information!
	state.Resources["foo"].ConnInfo = nil

	if !reflect.DeepEqual(actual, state) {
		t.Fatalf("bad: %#v", actual)
	}
}

// sensitiveState is used to store sensitive state information
// that should not be serialized. This is only used temporarily
// and is restored into the state.
type sensitiveState struct {
	ConnInfo map[string]map[string]string

	once sync.Once
}

func (s *sensitiveState) init() {
	s.once.Do(func() {
		s.ConnInfo = make(map[string]map[string]string)
	})
}

// testWriteStateV1 writes a state somewhere in a binary format.
// Only for testing now
func testWriteStateV1(d *StateV1, dst io.Writer) error {
	// Write the magic bytes so we can determine the file format later
	n, err := dst.Write([]byte(stateFormatMagic))
	if err != nil {
		return err
	}
	if n != len(stateFormatMagic) {
		return errors.New("failed to write state format magic bytes")
	}

	// Write a version byte so we can iterate on version at some point
	n, err = dst.Write([]byte{stateFormatVersion})
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("failed to write state version byte")
	}

	// Prevent sensitive information from being serialized
	sensitive := &sensitiveState{}
	sensitive.init()
	for name, r := range d.Resources {
		if r.ConnInfo != nil {
			sensitive.ConnInfo[name] = r.ConnInfo
			r.ConnInfo = nil
		}
	}

	// Serialize the state
	err = gob.NewEncoder(dst).Encode(d)

	// Restore the state
	for name, info := range sensitive.ConnInfo {
		d.Resources[name].ConnInfo = info
	}

	return err
}
