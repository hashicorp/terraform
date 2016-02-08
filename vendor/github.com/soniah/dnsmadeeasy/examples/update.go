package main

import (
	"fmt"
	dme "github.com/soniah/dnsmadeeasy"
	"log"
	"os"
)

func main() {
	akey := os.Getenv("DME_AKEY")
	skey := os.Getenv("DME_SKEY")
	domainID := os.Getenv("DME_DOMAINID")
	recordID := os.Getenv("DME_RECORDID")

	fmt.Println("Using these values:")
	fmt.Println("akey:", akey)
	fmt.Println("skey:", skey)
	fmt.Println("domainid:", domainID)
	fmt.Println("recordid:", recordID)

	if len(akey) == 0 || len(skey) == 0 || len(domainID) == 0 || len(recordID) == 0 {
		log.Fatalf("Environment variable(s) not set\n")
	}

	client, err := dme.NewClient(akey, skey)
	client.URL = dme.SandboxURL
	if err != nil {
		log.Fatalf("err: %v", err)
	}

	cr := map[string]interface{}{
		"name": "test-update",
	}

	result, err2 := client.UpdateRecord(domainID, recordID, cr)
	if err2 != nil {
		log.Fatalf("UpdateRecord result: %v error %v", result, err2)
	}
	log.Print("Result: ", result)
}
