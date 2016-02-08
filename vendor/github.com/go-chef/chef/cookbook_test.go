package chef

import (
	"fmt"
	"io/ioutil"
	"net/http"
	//"os"
	"testing"
)

const cookbookListResponseFile = "test/cookbooks_response.json"
const cookbookTestFile = "test/cookbook.json"

func TestCookbookList(t *testing.T) {
	setup()
	defer teardown()

	file, err := ioutil.ReadFile(cookbookListResponseFile)
	if err != nil {
		t.Error(err)
	}

	mux.HandleFunc("/cookbooks", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, string(file))
	})

	data, err := client.Cookbooks.List()
	if err != nil {
		t.Error(err)
	}

	if data == nil {
		t.Fatal("WTF we should have some data")
	}
	fmt.Println(data)

	_, err = client.Cookbooks.ListAvailableVersions("3")
	if err != nil {
		t.Error(err)
	}

	_, err = client.Cookbooks.ListAvailableVersions("0")
	if err != nil {
		t.Error(err)
	}
}

func TestCookbookListAvailableVersions_0(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/cookbooks", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "BAD FUCKING REQUEST", 503)
	})

	_, err := client.Cookbooks.ListAvailableVersions("2")
	if err == nil {
		t.Error("We expected this bad request to error", err)
	}
}

func TestCookBookDelete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/cookbooks/good", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "")
	})
	mux.HandleFunc("/cookbooks/bad", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", 404)
	})

	err := client.Cookbooks.Delete("bad", "1.1.1")
	if err == nil {
		t.Error("We expected this bad request to error", err)
	}

	err = client.Cookbooks.Delete("good", "1.1.1")
	if err != nil {
		t.Error(err)
	}
}

func TestCookBookGet(t *testing.T) {
	setup()
	defer teardown()

	cookbookVerionJSON := `{"url": "http://localhost:4000/cookbooks/apache2/5.1.0", "version": "5.1.0"}`
	mux.HandleFunc("/cookbooks/good", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, cookbookVerionJSON)
	})
	mux.HandleFunc("/cookbooks/bad", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", 404)
	})

	data, err := client.Cookbooks.Get("good")
	if err != nil {
		t.Error(err)
	}

	if data.Version != "5.1.0" {
		t.Errorf("We expected '5.1.0' and got '%s'\n", data.Version)
	}

	_, err = client.Cookbooks.Get("bad")
	if err == nil {
		t.Error("We expected this bad request to error", err)
	}
}

func TestCookBookGetAvailableVersions(t *testing.T) {
	setup()
	defer teardown()

	cookbookVerionsJSON := `
	{	"apache2": {
    "url": "http://localhost:4000/cookbooks/apache2",
    "versions": [
      {"url": "http://localhost:4000/cookbooks/apache2/5.1.0",
       "version": "5.1.0"},
      {"url": "http://localhost:4000/cookbooks/apache2/4.2.0",
       "version": "4.2.0"}
    ]
	}}`

	mux.HandleFunc("/cookbooks/good", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, cookbookVerionsJSON)
	})
	mux.HandleFunc("/cookbooks/bad", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", 404)
	})

	data, err := client.Cookbooks.GetAvailableVersions("good", "3")
	if err != nil {
		t.Error(err)
	}
	fmt.Println(data)

}
