package azure

import (
	"fmt"

	"github.com/Azure/go-autorest/autorest"
)

// it seems the cosmos API is not returning any sort of valid ID in the main response body
// so lets grab it from the response.request.url.path
func CosmosGetIDFromResponse(resp autorest.Response) (string, error) {
	if resp.Response == nil {
		return "", fmt.Errorf("Error: Unable to get Cosmos ID from Response: http response is nil")
	}

	if resp.Response.Request == nil {
		return "", fmt.Errorf("Error: Unable to get Cosmos ID from Response: Request is nil")
	}

	if resp.Response.Request.URL == nil {
		return "", fmt.Errorf("Error: Unable to get Cosmos ID from Response: URL is nil")
	}

	return resp.Response.Request.URL.Path, nil
}

type CosmosAccountID struct {
	ResourceID
	Account string
}

func ParseCosmosAccountID(id string) (*CosmosAccountID, error) {
	subid, err := ParseAzureResourceID(id)
	if err != nil {
		return nil, err
	}

	account, ok := subid.Path["databaseAccounts"]
	if !ok {
		return nil, fmt.Errorf("Error: Unable to parse Cosmos Database Resource ID: databaseAccounts is missing from: %s", id)
	}

	return &CosmosAccountID{
		ResourceID: *subid,
		Account:    account,
	}, nil
}

type CosmosDatabaseID struct {
	CosmosAccountID
	Database string
}

func ParseCosmosDatabaseID(id string) (*CosmosDatabaseID, error) {
	subid, err := ParseCosmosAccountID(id)
	if err != nil {
		return nil, err
	}

	db, ok := subid.Path["databases"]
	if !ok {
		return nil, fmt.Errorf("Error: Unable to parse Cosmos Database Resource ID: databases is missing from: %s", id)
	}

	return &CosmosDatabaseID{
		CosmosAccountID: *subid,
		Database:        db,
	}, nil
}

type CosmosDatabaseCollectionID struct {
	CosmosDatabaseID
	Collection string
}

func ParseCosmosDatabaseCollectionID(id string) (*CosmosDatabaseCollectionID, error) {
	subid, err := ParseCosmosDatabaseID(id)
	if err != nil {
		return nil, err
	}

	collection, ok := subid.Path["collections"]
	if !ok {
		return nil, fmt.Errorf("Error: Unable to parse Cosmos Database Resource ID: collections is missing from: %s", id)
	}

	return &CosmosDatabaseCollectionID{
		CosmosDatabaseID: *subid,
		Collection:       collection,
	}, nil
}

type CosmosKeyspaceID struct {
	CosmosAccountID
	Keyspace string
}

func ParseCosmosKeyspaceID(id string) (*CosmosKeyspaceID, error) {
	subid, err := ParseCosmosAccountID(id)
	if err != nil {
		return nil, err
	}

	ks, ok := subid.Path["keyspaces"]
	if !ok {
		return nil, fmt.Errorf("Error: Unable to parse Cosmos Keyspace Resource ID: keyspaces is missing from: %s", id)
	}

	return &CosmosKeyspaceID{
		CosmosAccountID: *subid,
		Keyspace:        ks,
	}, nil
}

type CosmosTableID struct {
	CosmosAccountID
	Table string
}

func ParseCosmosTableID(id string) (*CosmosTableID, error) {
	subid, err := ParseCosmosAccountID(id)
	if err != nil {
		return nil, err
	}

	table, ok := subid.Path["tables"]
	if !ok {
		return nil, fmt.Errorf("Error: Unable to parse Cosmos Table Resource ID: tables is missing from: %s", id)
	}

	return &CosmosTableID{
		CosmosAccountID: *subid,
		Table:           table,
	}, nil
}
