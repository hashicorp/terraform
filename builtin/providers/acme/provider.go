package acme

import (
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/terraform"
)

// Provider returns the terraform.ResourceProvider structure for the ACME
// provider.
func Provider() terraform.ResourceProvider {
	return &schema.Provider{
		ResourcesMap: map[string]*schema.Resource{
			"acme_registration": resourceACMERegistration(),
			"acme_certificate":  resourceACMECertificate(),
		},
	}
}
