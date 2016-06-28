package examples

import (
	"fmt"
	"log"

	"github.com/Azure/azure-sdk-for-go/arm/examples/helpers"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
)

func createAccount(resourceGroup, name string) {
	c, err := helpers.LoadCredentials()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	ac := storage.NewAccountsClient(c["subscriptionID"])

	spt, err := helpers.NewServicePrincipalTokenFromCredentials(c, azure.PublicCloud.ResourceManagerEndpoint)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	ac.Authorizer = spt

	cna, err := ac.CheckNameAvailability(
		storage.AccountCheckNameAvailabilityParameters{
			Name: to.StringPtr(name),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts")})
	if err != nil {
		log.Fatalf("Error: %v", err)
		return
	}
	if !to.Bool(cna.NameAvailable) {
		fmt.Printf("%s is unavailable -- try again\n", name)
		return
	}
	fmt.Printf("%s is available\n\n", name)

	cp := storage.AccountCreateParameters{}
	cp.Location = to.StringPtr("westus")
	cp.Properties = &storage.AccountPropertiesCreateParameters{AccountType: storage.StandardLRS}

	cancel := make(chan struct{})
	_, err = ac.Create(resourceGroup, name, cp, cancel)
	if err != nil {
		fmt.Printf("Create failed: %v\n", err)
		return
	}

	fmt.Printf("Successfully created %s.%s\n\n", resourceGroup, name)

	r, err := ac.Delete(resourceGroup, name)
	if err != nil {
		fmt.Printf("Delete of %s.%s failed with status %s\n...%v\n", resourceGroup, name, r.Status, err)
		return
	}
	fmt.Printf("Deletion of %s.%s succeeded -- %s\n", resourceGroup, name, r.Status)
}
