package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/go-chef/chef"
)

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

	// List Indexes
	indexes, err := client.Search.Indexes()
	if err != nil {
		log.Fatal("Couldn't list nodes: ", err)
	}

	// dump the Index list in Json
	jsonData, err := json.MarshalIndent(indexes, "", "\t")
	os.Stdout.Write(jsonData)
	os.Stdout.WriteString("\n")

	// build a seach query
	query, err := client.Search.NewQuery("node", "name:*")
	if err != nil {
		log.Fatal("Error building query ", err)
	}

	// Run the query
	res, err := query.Do(client)
	if err != nil {
		log.Fatal("Error running query ", err)
	}

	// <3 spew
	spew.Dump(res)

	// dump out results back in json for fun
	jsonData, err = json.MarshalIndent(res, "", "\t")
	os.Stdout.Write(jsonData)
	os.Stdout.WriteString("\n")

	// You can also use the service to run a query
	res, err = client.Search.Exec("node", "name:*")
	if err != nil {
		log.Fatal("Error running Search.Exec() ", err)
	}

	// dump out results back in json for fun
	jsonData, err = json.MarshalIndent(res, "", "\t")
	os.Stdout.Write(jsonData)
	os.Stdout.WriteString("\n")

	// Partial search
	log.Print("Partial Search")
	part := make(map[string]interface{})
	part["name"] = []string{"name"}
	pres, err := client.Search.PartialExec("node", "*:*", part)
	if err != nil {
		log.Fatal("Error running Search.PartialExec()", err)
	}

	jsonData, err = json.MarshalIndent(pres, "", "\t")
	os.Stdout.Write(jsonData)
	os.Stdout.WriteString("\n")

}
