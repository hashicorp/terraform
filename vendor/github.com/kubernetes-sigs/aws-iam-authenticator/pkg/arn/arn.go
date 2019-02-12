package arn

import (
	"fmt"
	"strings"

	awsarn "github.com/aws/aws-sdk-go/aws/arn"
	"github.com/aws/aws-sdk-go/aws/endpoints"
)

// Canonicalize validates IAM resources are appropriate for the authenticator
// and converts STS assumed roles into the IAM role resource.
//
// Supported IAM resources are:
//   * AWS account: arn:aws:iam::123456789012:root
//   * IAM user: arn:aws:iam::123456789012:user/Bob
//   * IAM role: arn:aws:iam::123456789012:role/S3Access
//   * IAM Assumed role: arn:aws:sts::123456789012:assumed-role/Accounting-Role/Mary (converted to IAM role)
//   * Federated user: arn:aws:sts::123456789012:federated-user/Bob
func Canonicalize(arn string) (string, error) {
	parsed, err := awsarn.Parse(arn)
	if err != nil {
		return "", fmt.Errorf("arn '%s' is invalid: '%v'", arn, err)
	}

	if err := checkPartition(parsed.Partition); err != nil {
		return "", fmt.Errorf("arn '%s' does not have a recognized partition", arn)
	}

	parts := strings.Split(parsed.Resource, "/")
	resource := parts[0]

	switch parsed.Service {
	case "sts":
		switch resource {
		case "federated-user":
			return arn, nil
		case "assumed-role":
			if len(parts) < 3 {
				return "", fmt.Errorf("assumed-role arn '%s' does not have a role", arn)
			}
			// IAM ARNs can contain paths, part[0] is resource, parts[len(parts)] is the SessionName.
			role := strings.Join(parts[1:len(parts)-1], "/")
			return fmt.Sprintf("arn:%s:iam::%s:role/%s", parsed.Partition, parsed.AccountID, role), nil
		default:
			return "", fmt.Errorf("unrecognized resource %s for service sts", parsed.Resource)
		}
	case "iam":
		switch resource {
		case "role", "user", "root":
			return arn, nil
		default:
			return "", fmt.Errorf("unrecognized resource %s for service iam", parsed.Resource)
		}
	}

	return "", fmt.Errorf("service %s in arn %s is not a valid service for identities", parsed.Service, arn)
}

func checkPartition(partition string) error {
	switch partition {
	case endpoints.AwsPartitionID:
	case endpoints.AwsCnPartitionID:
	case endpoints.AwsUsGovPartitionID:
	default:
		return fmt.Errorf("partion %s is not recognized", partition)
	}
	return nil
}
