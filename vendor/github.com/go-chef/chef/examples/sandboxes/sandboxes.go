package main

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/cenkalti/backoff"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chef/chef"
)

//random_data makes random byte slice for building junk sandbox data
func random_data(size int) (b []byte) {
	b = make([]byte, size)
	rand.Read(b)
	return
}

func main() {
	// read a client key
	key, err := ioutil.ReadFile("key.pem")
	if err != nil {
		fmt.Println("Couldn't read key.pem:", err)
		os.Exit(1)
	}

	// build a client
	client, err := chef.NewClient(&chef.Config{
		Name: "foo",
		Key:  string(key),
		// goiardi is on port 4545 by default. chef-zero is 8889
		BaseURL: "http://localhost:4545",
	})
	if err != nil {
		fmt.Println("Issue setting up client:", err)
		os.Exit(1)
	}

	// create junk files and sums
	files := make(map[string][]byte)
	sums := make([]string, 10)
	for i := 0; i < 10; i++ {
		data := random_data(1024)
		hashstr := fmt.Sprintf("%x", md5.Sum(data))
		files[hashstr] = data
		sums[i] = hashstr
	}

	// post the new sums/files to the sandbox
	postResp, err := client.Sandboxes.Post(sums)
	if err != nil {
		fmt.Println("error making request: ", err)
		os.Exit(1)
	}

	// Dump the the server response data
	j, err := json.MarshalIndent(postResp, "", "    ")
	fmt.Printf("%s", j)

	// Let's upload the files that postRep thinks we should upload
	for hash, item := range postResp.Checksums {
		if item.Upload == true {
			if hash == "" {
				continue
			}
			// If you were writing this in your own tool you could just use the FH and let the Reader interface suck out the content instead of doing the convert.
			fmt.Printf("\nUploading: %s --->  %v\n\n", hash, item)
			req, err := client.NewRequest("PUT", item.Url, bytes.NewReader(files[hash]))
			if err != nil {
				fmt.Println("This shouldn't happen:", err.Error())
				os.Exit(1)
			}

			// post the files
			upload := func() error {
				_, err = client.Do(req, nil)
				return err
			}

			// with exp backoff !
			err = backoff.Retry(upload, backoff.NewExponentialBackOff())
			if err != nil {
				fmt.Println("error posting files to the sandbox: ", err.Error())
			}
		}
	}

	// Now lets tell the server we have uploaded all the things.
	sandbox, err := client.Sandboxes.Put(postResp.ID)
	if err != nil {
		fmt.Println("Error commiting sandbox: ", err.Error())
		os.Exit(1)
	}

	// and heres yer commited sandbox
	spew.Dump(sandbox)

}
