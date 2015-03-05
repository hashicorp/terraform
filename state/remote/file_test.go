package remote

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestFileClient_impl(t *testing.T) {
	var _ Client = new(FileClient)
}

func TestFileClient(t *testing.T) {
	tf, err := ioutil.TempFile("", "tf")
	if err != nil {
		t.Fatalf("err: %s", err)
	}
	tf.Close()
	defer os.Remove(tf.Name())

	client, err := fileFactory(map[string]string{
		"path": tf.Name(),
	})
	if err != nil {
		t.Fatalf("bad: %s", err)
	}

	testClient(t, client)
}
