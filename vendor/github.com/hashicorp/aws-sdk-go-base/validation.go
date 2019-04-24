package awsbase

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/endpoints"
)

// ValidateAccountID checks if the given AWS account ID is specifically allowed or forbidden.
// The allowedAccountIDs can be used as a whitelist and forbiddenAccountIDs can be used as a blacklist.
func ValidateAccountID(accountID string, allowedAccountIDs, forbiddenAccountIDs []string) error {
	if len(forbiddenAccountIDs) > 0 {
		for _, forbiddenAccountID := range forbiddenAccountIDs {
			if accountID == forbiddenAccountID {
				return fmt.Errorf("Forbidden AWS Account ID: %s", accountID)
			}
		}
	}

	if len(allowedAccountIDs) > 0 {
		for _, allowedAccountID := range allowedAccountIDs {
			if accountID == allowedAccountID {
				return nil
			}
		}

		return fmt.Errorf("AWS Account ID not allowed: %s)", accountID)
	}

	return nil
}

// ValidateRegion checks if the given region is a valid AWS region.
func ValidateRegion(region string) error {
	for _, partition := range endpoints.DefaultPartitions() {
		for _, partitionRegion := range partition.Regions() {
			if region == partitionRegion.ID() {
				return nil
			}
		}
	}

	return fmt.Errorf("Invalid AWS Region: %s", region)
}
