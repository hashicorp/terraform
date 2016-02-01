package chef

import (
	"io"
	"io/ioutil"
	"os"
	"testing"
)

type TestEncoder struct {
	Name       string
	Awesome    []string
	OtherStuff map[string]string
}

func TestEncoderJSONReader(t *testing.T) {
	f, err := ioutil.TempFile("test/", "reader")
	if err != nil {
		t.Error(err)
	}

	defer f.Close()
	defer os.Remove(f.Name())

	tr := &TestEncoder{
		Name:    "Test Reader",
		Awesome: []string{"foo", "bar", "baz"},
		OtherStuff: map[string]string{
			"foo": "bar",
			"baz": "banana",
		},
	}

	// Generate body
	body, err := JSONReader(tr)
	if err != nil {
		t.Error(err)
	}

	t.Log(body)

	_, err = io.Copy(f, body)
	if err != nil {
		t.Error(err)
	}
}
