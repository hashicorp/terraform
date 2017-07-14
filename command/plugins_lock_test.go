package command

import (
	"io/ioutil"
	"reflect"
	"testing"
)

func TestPluginSHA256LockFile(t *testing.T) {
	f, err := ioutil.TempFile("", "tf-pluginsha1lockfile-test-")
	if err != nil {
		t.Fatalf("failed to create temporary file: %s", err)
	}
	f.Close()
	//defer os.Remove(f.Name())
	t.Logf("working in %s", f.Name())

	plf := &pluginSHA256LockFile{
		Filename: f.Name(),
	}

	// Initially the file is invalid, so we should get an empty map.
	digests := plf.Read()
	if !reflect.DeepEqual(digests, map[string][]byte{}) {
		t.Errorf("wrong initial content %#v; want empty map", digests)
	}

	digests = map[string][]byte{
		"test": []byte("hello world"),
	}
	err = plf.Write(digests)
	if err != nil {
		t.Fatalf("failed to write lock file: %s", err)
	}

	got := plf.Read()
	if !reflect.DeepEqual(got, digests) {
		t.Errorf("wrong content %#v after write; want %#v", got, digests)
	}
}
