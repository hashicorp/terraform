package chef

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"net/http"
	_ "reflect"
	"testing"
	. "github.com/smartystreets/goconvey/convey"
)

// generate random data for sandbox
func random_data(size int) (b []byte) {
	b = make([]byte, size)
	rand.Read(b)
	return
}

//	mux.HandleFunc("/sandboxes/f1c560ccb472448e9cfb31ff98134247", func(w http.ResponseWriter, r *http.Request) { })
func TestSandboxesPost(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/sandboxes", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"sandbox_id": "f1c560ccb472448e9cfb31ff98134247",
			"uri": "http://trendy.local:4545/sandboxes/f1c560ccb472448e9cfb31ff98134247",
			"Checksums": {
					"4bd9946774fff1fb53745c645e447c9d20a14cac410b9eea037299247e70aa1e": {
							"url": "http://trendy.local:4545/file_store/4bd9946774fff1fb53745c645e447c9d20a14cac410b9eea037299247e70aa1e",
							"needs_upload": true
					},
					"548c6928e3f5a800a0e9cc146647a31a2353c42950a611cfca646819cdaa54fa": {
							"url": "http://trendy.local:4545/file_store/548c6928e3f5a800a0e9cc146647a31a2353c42950a611cfca646819cdaa54fa",
							"needs_upload": true
			}}}`)
	})

	// create junk files and sums
	files := make(map[string][]byte)
	// slice of strings for holding our hashes
	sums := make([]string, 10)
	for i := 0; i <= 10; i++ {
		data := random_data(1024)
		hashstr := fmt.Sprintf("%x", md5.Sum(data))
		files[hashstr] = data
		sums = append(sums, hashstr)
	}

	// post the new sums/files to the sandbox
	_, err := client.Sandboxes.Post(sums)
	if err != nil {
		t.Errorf("Snadbox Post error making request: ", err)
	}
}

func TestSandboxesPut(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/sandboxes/f1c560ccb472448e9cfb31ff98134247", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"guid": "123",
			"name": "123",
			"CreateionTime": "2014-08-23 18:13:37",
			"is_completed": true,
			"uri": "https://127.0.0.1/sandboxes/f1c560ccb472448e9cfb31ff98134247",
	  	"checksums": [
				"3124216defe5849089a577ffefb0bb05",
				"e06ddfa07ca97c80c368d08e189b928a",
				"eb65444f8adeb11c56cf0df201a07cb4"
			]
		}`)
	})

	sandbox, err := client.Sandboxes.Put("f1c560ccb472448e9cfb31ff98134247")
	if err != nil {
		t.Errorf("Snadbox Put error making request: ", err)
	}

	expected := Sandbox{
		ID:        "123",
		Name:      "123",
		Completed: true,
		Checksums: []string{
			"3124216defe5849089a577ffefb0bb05",
			"e06ddfa07ca97c80c368d08e189b928a",
			"eb65444f8adeb11c56cf0df201a07cb4",
		},
	}

	Convey("Sandbox Equality", t, func() {
		So(sandbox, ShouldResemble, expected)
	})
}
