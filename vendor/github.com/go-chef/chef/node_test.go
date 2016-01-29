package chef

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"reflect"
	"testing"
)

var (
	testNodeJSON = "test/node.json"
)

func TestNodeFromJSONDecoder(t *testing.T) {
	if file, err := os.Open(testNodeJSON); err != nil {
		t.Error("unexpected error", err, "during os.Open on", testNodeJSON)
	} else {
		dec := json.NewDecoder(file)
		var n Node
		if err := dec.Decode(&n); err == io.EOF {
			log.Println(n)
		} else if err != nil {
			log.Fatal(err)
		}
	}
}

func TestNode_NewNode(t *testing.T) {
	n := NewNode("testnode")
	expect := Node{
		Name:        "testnode",
		Environment: "_default",
		ChefType:    "node",
		JsonClass:   "Chef::Node",
	}

	if !reflect.DeepEqual(n, expect) {
		t.Errorf("NewNode returned %+v, want %+v", n, expect)
	}
}

func TestNodesService_Methods(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/nodes", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET":
			fmt.Fprintf(w, `{"node1":"https://chef/nodes/node1", "node2":"https://chef/nodes/node2"}`)
		case r.Method == "POST":
			fmt.Fprintf(w, `{ "uri": "http://localhost:4545/nodes/node1" }`)
		}
	})

	mux.HandleFunc("/nodes/node1", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "GET" || r.Method == "PUT":
			fmt.Fprintf(w, `{
	    "name": "node1",
	    "json_class": "Chef::Node",
	    "chef_type": "node",
	    "chef_environment": "development"
		}`)
		case r.Method == "DELETE":
		}
	})

	// Test list
	nodes, err := client.Nodes.List()
	if err != nil {
		t.Errorf("Nodes.List returned error: %v", err)
	}

	listWant := map[string]string{"node1": "https://chef/nodes/node1", "node2": "https://chef/nodes/node2"}

	if !reflect.DeepEqual(nodes, listWant) {
		t.Errorf("Nodes.List returned %+v, want %+v", nodes, listWant)
	}

	// test Get
	node, err := client.Nodes.Get("node1")
	if err != nil {
		t.Errorf("Nodes.Get returned error: %v", err)
	}

	wantNode := NewNode("node1")
	wantNode.Environment = "development"
	if !reflect.DeepEqual(node, wantNode) {
		t.Errorf("Nodes.Get returned %+v, want %+v", node, wantNode)
	}

	// test Post
	res, err := client.Nodes.Post(wantNode)
	if err != nil {
		t.Errorf("Nodes.Post returned error: %s", err.Error())
	}

	postResult := &NodeResult{"http://localhost:4545/nodes/node1"}
	if !reflect.DeepEqual(postResult, res) {
		t.Errorf("Nodes.Post returned %+v, want %+v", res, postResult)
	}

	// test Put
	putRes, err := client.Nodes.Put(node)
	if err != nil {
		t.Errorf("Nodes.Put returned error", err)
	}

	if !reflect.DeepEqual(putRes, node) {
		t.Errorf("Nodes.Post returned %+v, want %+v", putRes, node)
	}

	// test Delete
	err = client.Nodes.Delete(node.Name)
	if err != nil {
		t.Errorf("Nodes.Delete returned error", err)
	}
}
