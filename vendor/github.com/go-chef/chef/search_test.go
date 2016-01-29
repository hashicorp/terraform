package chef

import (
	"fmt"
	"net/http"
	"reflect"
	"testing"
)

func TestSearch_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"node": "http://localhost:4000/search/node", 
			"role": "http://localhost:4000/search/role", 
			"client": "http://localhost:4000/search/client", 
			"users": "http://localhost:4000/search/users" 
		}`)
	})

	indexes, err := client.Search.Indexes()
	if err != nil {
		t.Errorf("Search.Get returned error: %+v", err)
	}
	wantedIdx := map[string]string{
		"node":   "http://localhost:4000/search/node",
		"role":   "http://localhost:4000/search/role",
		"client": "http://localhost:4000/search/client",
		"users":  "http://localhost:4000/search/users",
	}
	if !reflect.DeepEqual(indexes, wantedIdx) {
		t.Errorf("Search.Get returned %+v, want %+v", indexes, wantedIdx)
	}
}

func TestSearch_ExecDo(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/search/nodes", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
	    "total": 1,
	    "start": 0,
	    "rows": [
	       {
	        "overrides": {"hardware_type": "laptop"},
	        "name": "latte",
	        "chef_type": "node",
	        "json_class": "Chef::Node",
	        "attributes": {"hardware_type": "laptop"},
	        "run_list": ["recipe[unicorn]"],
	        "defaults": {}
	       }
				 ]
		}`)
	})

	// test the fail case
	_, err := client.Search.NewQuery("foo", "failsauce")
	if err == nil {
		t.Errorf("Bad query wasn't caught")
	}

	// test the fail case
	_, err = client.Search.Exec("foo", "failsauce")
	if err == nil {
		t.Errorf("Bad query wasn't caught")
	}

	// test the positive case
	query, err := client.Search.NewQuery("nodes", "name:latte")
	if err != nil {
		t.Errorf("failed to create query")
	}

	// for now we aren't testing the result..
	_, err = query.Do(client)
	if err != nil {
		t.Errorf("Search.Exec failed", err)
	}

	_, err = client.Search.Exec("nodes", "name:latte")
	if err != nil {
		t.Errorf("Search.Exec failed", err)
	}

}
