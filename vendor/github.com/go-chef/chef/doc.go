/*
This is a chef server api client.
This Library can be used to write tools to interact with the chef server.

The testing can be run with `go test`, and the client can be used as per normal via `go get github.com/go-chef/chef`
Documentation can be found on GoDoc at  http://godoc.org/github.com/go-chef/chef

This is example code generating a new node on the chef-server.


		package main

		import (
			"encoding/json"
			"fmt"
			"io/ioutil"
			"log"
			"os"

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

			// Create a Node object
			// TOOD: should have a constructor for this
			ranjib := chef.Node{
				Name:        "ranjib",
				Environment: "_default",
				ChefType:    "node",
				JsonClass:   "Chef::Node",
				RunList:     []string{"pwn"},
			}

			// Create
			_, err = client.Nodes.Post(ranjib)
			if err != nil {
				log.Fatal("Couldn't create node. ", err)
			}

			// List nodes
			nodeList, err := client.Nodes.List()
			if err != nil {
				log.Fatal("Couldn't list nodes: ", err)
			}

			// dump the node list in Json
			jsonData, err := json.MarshalIndent(nodeList, "", "\t")
			os.Stdout.Write(jsonData)
			os.Stdout.WriteString("\n")

			// dump the ranjib node we got from server in JSON!
			serverNode, _ := client.Nodes.Get("ranjib")
			if err != nil {
				log.Fatal("Couldn't get node: ", err)
			}
			jsonData, err = json.MarshalIndent(serverNode, "", "\t")
			os.Stdout.Write(jsonData)
			os.Stdout.WriteString("\n")

			// update node
			ranjib.RunList = append(ranjib.RunList, "recipe[works]")
			jsonData, err = json.MarshalIndent(ranjib, "", "\t")
			os.Stdout.Write(jsonData)
			os.Stdout.WriteString("\n")

			_, err = client.Nodes.Put(ranjib)
			if err != nil {
				log.Fatal("Couldn't update node: ", err)
			}

			// Delete node ignoring errors :)
			client.Nodes.Delete(ranjib.Name)

		}

*/
package chef
