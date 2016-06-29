package examples

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/Azure/azure-sdk-for-go/arm/examples/helpers"
	"github.com/Azure/azure-sdk-for-go/arm/storage"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/to"
)

func withInspection() autorest.PrepareDecorator {
	return func(p autorest.Preparer) autorest.Preparer {
		return autorest.PreparerFunc(func(r *http.Request) (*http.Request, error) {
			fmt.Printf("Inspecting Request: %s %s\n", r.Method, r.URL)
			return p.Prepare(r)
		})
	}
}

func byInspecting() autorest.RespondDecorator {
	return func(r autorest.Responder) autorest.Responder {
		return autorest.ResponderFunc(func(resp *http.Response) error {
			fmt.Printf("Inspecting Response: %s for %s %s\n", resp.Status, resp.Request.Method, resp.Request.URL)
			return r.Respond(resp)
		})
	}
}

func checkName(name string) {
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

	ac.Sender = autorest.CreateSender(
		autorest.WithLogging(log.New(os.Stdout, "sdk-example: ", log.LstdFlags)))

	ac.RequestInspector = withInspection()
	ac.ResponseInspector = byInspecting()
	cna, err := ac.CheckNameAvailability(
		storage.AccountCheckNameAvailabilityParameters{
			Name: to.StringPtr(name),
			Type: to.StringPtr("Microsoft.Storage/storageAccounts")})

	if err != nil {
		log.Fatalf("Error: %v", err)
	} else {
		if to.Bool(cna.NameAvailable) {
			fmt.Printf("The name '%s' is available\n", name)
		} else {
			fmt.Printf("The name '%s' is unavailable because %s\n", name, to.String(cna.Message))
		}
	}
}
