// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package command

import (
	"io/ioutil"
	"os"
	"reflect"
	"testing"
)

func TestPluginSHA256LockFile_Read(t *testing.T) {
	f, err := ioutil.TempFile(t.TempDir(), "tf-pluginsha1lockfile-test-")
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}
	f.Close()
	defer os.Remove(f.Name())

	plf := &pluginSHA256LockFile{
		Filename: f.Name(),
	}

	// Initially the file is invalid, so we should get an empty map.
	digests := plf.Read()
	if !reflect.DeepEqual(digests, map[string][]byte{}) {
		t.Errorf("wrong initial content %#v; want empty map", digests)
	}
}
