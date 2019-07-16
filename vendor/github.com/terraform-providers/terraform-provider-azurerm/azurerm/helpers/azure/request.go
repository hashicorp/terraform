package azure

import (
	"log"
	"sync"

	"github.com/Azure/go-autorest/autorest"
	"github.com/hashicorp/go-uuid"
)

const (
	// HeaderCorrelationRequestID is the Azure extension header to set a user-specified correlation request ID.
	HeaderCorrelationRequestID = "x-ms-correlation-request-id"
)

var (
	msCorrelationRequestIDOnce sync.Once
	msCorrelationRequestID     string
)

// WithCorrelationRequestID returns a PrepareDecorator that adds an HTTP extension header of
// `x-ms-correlation-request-id` whose value is passed, undecorated UUID (e.g.,
// `7F5A6223-F475-4A9C-B9D5-12575AA6B11B`).
func WithCorrelationRequestID(uuid string) autorest.PrepareDecorator {
	return autorest.WithHeader(HeaderCorrelationRequestID, uuid)
}

// CorrelationRequestID generates an UUID to pass through `x-ms-correlation-request-id` header.
func CorrelationRequestID() string {
	msCorrelationRequestIDOnce.Do(func() {
		var err error
		msCorrelationRequestID, err = uuid.GenerateUUID()

		if err != nil {
			log.Printf("[WARN] Fail to generate uuid for msCorrelationRequestID: %+v", err)
		}
	})

	log.Printf("[DEBUG] AzureRM Correlation Request Id: %s", msCorrelationRequestID)
	return msCorrelationRequestID
}
