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
	testClientJSON = "test/client.json"
)

func TestClientFromJSONDecoder(t *testing.T) {
	if file, err := os.Open(testClientJSON); err != nil {
		t.Error("unexpected error", err, "during os.Open on", testClientJSON)
	} else {
		dec := json.NewDecoder(file)
		var n Client
		if err := dec.Decode(&n); err == io.EOF {
			log.Println(n)
		} else if err != nil {
			log.Fatal(err)
		}
	}
}

func TestClientsService_List(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/clients", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"client1": "http://localhost/clients/client1", "client2": "http://localhost/clients/client2"}`)
	})

	response, err := client.Clients.List()
	if err != nil {
		t.Errorf("Clients.List returned error: %v", err)
	}

	want := "client1 => http://localhost/clients/client1\nclient2 => http://localhost/clients/client2\n"
	if response.String() != want {
		t.Errorf("Clients.List returned:\n%+v\nwant:\n%+v\n", response.String(), want)
	}
}

func TestClientsService_Get(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/clients/client1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
      "clientname": "client1",
      "orgname": "org_name",
      "validator": false,
      "certificate": "-----BEGIN CERTIFICATE-----",
      "name": "node_name"
    }`)
	})

	_, err := client.Clients.Get("client1")
	if err != nil {
		t.Errorf("Clients.Get returned error: %v", err)
	}
}

func TestClientsService_Create(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/clients", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"uri": "http://localhost/clients/client", "private_key": "-----BEGIN PRIVATE KEY-----"}`)
	})

	response, err := client.Clients.Create("client", false)
	if err != nil {
		t.Errorf("Clients.Create returned error: %v", err)
	}

	want := &ApiClientCreateResult{Uri: "http://localhost/clients/client", PrivateKey: "-----BEGIN PRIVATE KEY-----"}
	if !reflect.DeepEqual(response, want) {
		t.Errorf("Clients.Create returned %+v, want %+v", response, want)
	}
}

func TestClientsService_Delete(t *testing.T) {
	setup()
	defer teardown()

	mux.HandleFunc("/clients/client1", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"name": "client1", "json_class": "Chef::Client", "chef_type": "client"}`)
	})

	err := client.Clients.Delete("client1")
	if err != nil {
		t.Errorf("Clients.Delete returned error: %v", err)
	}
}
