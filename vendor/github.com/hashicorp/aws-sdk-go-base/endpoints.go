package awsbase

import (
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
)

func (c *Config) EndpointResolver() endpoints.Resolver {
	resolver := func(service, region string, optFns ...func(*endpoints.Options)) (endpoints.ResolvedEndpoint, error) {
		// Ensure we pass all existing information (e.g. SigningRegion) and
		// only override the URL, otherwise a MissingRegion error can occur
		// when aws.Config.Region is not defined.
		resolvedEndpoint, err := endpoints.DefaultResolver().EndpointFor(service, region, optFns...)

		if err != nil {
			return resolvedEndpoint, err
		}

		switch service {
		case ec2metadata.ServiceName:
			if endpoint := os.Getenv("AWS_METADATA_URL"); endpoint != "" {
				log.Printf("[INFO] Setting custom EC2 metadata endpoint: %s", endpoint)
				resolvedEndpoint.URL = endpoint
			}
		case iam.ServiceName:
			if endpoint := c.IamEndpoint; endpoint != "" {
				log.Printf("[INFO] Setting custom IAM endpoint: %s", endpoint)
				resolvedEndpoint.URL = endpoint
			}
		case sts.ServiceName:
			if endpoint := c.StsEndpoint; endpoint != "" {
				log.Printf("[INFO] Setting custom STS endpoint: %s", endpoint)
				resolvedEndpoint.URL = endpoint
			}
		}

		return resolvedEndpoint, nil
	}

	return endpoints.ResolverFunc(resolver)
}
