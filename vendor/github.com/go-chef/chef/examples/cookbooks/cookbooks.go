package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/go-chef/chef"
)

func main() {
	// simple arg parsing
	//cookPath := flag.String("cookbook", "c", "Path to cookbook for upload")
	//	flag.Parse()

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
		BaseURL: "http://localhost:8443",
	})
	if err != nil {
		fmt.Println("Issue setting up client:", err)
		os.Exit(1)
	}

	// List Cookbooks
	cookList, err := client.Cookbooks.List()
	if err != nil {
		fmt.Println("Issue listing cookbooks:", err)
	}

	// Print out the list
	fmt.Println(cookList)
	/*
			*'parse' metadata...
		   this would prefer .json over .rb
		    if it's .rb lets maybe try to eval it ?
		      otherwise just extract name/version if they exist
	*/

	/*


		  	* generate sums
				* create sandbox
				* upload to sandbox
				*
	*/

}
