package aws

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awsCredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/hashicorp/go-cleanhttp"
)

func GetAccountId(iamconn *iam.IAM, stsconn *sts.STS, authProviderName string) (string, error) {
	// If we have creds from instance profile, we can use metadata API
	if authProviderName == ec2rolecreds.ProviderName {
		log.Println("[DEBUG] Trying to get account ID via AWS Metadata API")

		cfg := &aws.Config{}
		setOptionalEndpoint(cfg)
		metadataClient := ec2metadata.New(session.New(cfg))
		info, err := metadataClient.IAMInfo()
		if err != nil {
			// This can be triggered when no IAM Role is assigned
			// or AWS just happens to return invalid response
			return "", fmt.Errorf("Failed getting EC2 IAM info: %s", err)
		}

		return parseAccountIdFromArn(info.InstanceProfileArn)
	}

	// Then try IAM GetUser
	log.Println("[DEBUG] Trying to get account ID via iam:GetUser")
	outUser, err := iamconn.GetUser(nil)
	if err == nil {
		return parseAccountIdFromArn(*outUser.User.Arn)
	}

	awsErr, ok := err.(awserr.Error)
	// AccessDenied and ValidationError can be raised
	// if credentials belong to federated profile, so we ignore these
	if !ok || (awsErr.Code() != "AccessDenied" && awsErr.Code() != "ValidationError") {
		return "", fmt.Errorf("Failed getting account ID via 'iam:GetUser': %s", err)
	}
	log.Printf("[DEBUG] Getting account ID via iam:GetUser failed: %s", err)

	// Then try STS GetCallerIdentity
	log.Println("[DEBUG] Trying to get account ID via sts:GetCallerIdentity")
	outCallerIdentity, err := stsconn.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err == nil {
		return *outCallerIdentity.Account, nil
	}
	log.Printf("[DEBUG] Getting account ID via sts:GetCallerIdentity failed: %s", err)

	// Then try IAM ListRoles
	log.Println("[DEBUG] Trying to get account ID via iam:ListRoles")
	outRoles, err := iamconn.ListRoles(&iam.ListRolesInput{
		MaxItems: aws.Int64(int64(1)),
	})
	if err != nil {
		return "", fmt.Errorf("Failed getting account ID via 'iam:ListRoles': %s", err)
	}

	if len(outRoles.Roles) < 1 {
		return "", fmt.Errorf("Failed getting account ID via 'iam:ListRoles': No roles available")
	}

	return parseAccountIdFromArn(*outRoles.Roles[0].Arn)
}

func parseAccountIdFromArn(arn string) (string, error) {
	parts := strings.Split(arn, ":")
	if len(parts) < 5 {
		return "", fmt.Errorf("Unable to parse ID from invalid ARN: %q", arn)
	}
	return parts[4], nil
}

// This function is responsible for reading credentials from the
// environment in the case that they're not explicitly specified
// in the Terraform configuration.
func GetCredentials(key, secret, token, profile, credsfile string) *awsCredentials.Credentials {
	// build a chain provider, lazy-evaulated by aws-sdk
	providers := []awsCredentials.Provider{
		&awsCredentials.StaticProvider{Value: awsCredentials.Value{
			AccessKeyID:     key,
			SecretAccessKey: secret,
			SessionToken:    token,
		}},
		&awsCredentials.EnvProvider{},
		&awsCredentials.SharedCredentialsProvider{
			Filename: credsfile,
			Profile:  profile,
		},
	}

	// Build isolated HTTP client to avoid issues with globally-shared settings
	client := cleanhttp.DefaultClient()

	// Keep the timeout low as we don't want to wait in non-EC2 environments
	client.Timeout = 100 * time.Millisecond
	cfg := &aws.Config{
		HTTPClient: client,
	}
	usedEndpoint := setOptionalEndpoint(cfg)

	// Real AWS should reply to a simple metadata request.
	// We check it actually does to ensure something else didn't just
	// happen to be listening on the same IP:Port
	metadataClient := ec2metadata.New(session.New(cfg))
	if metadataClient.Available() {
		providers = append(providers, &ec2rolecreds.EC2RoleProvider{
			Client: metadataClient,
		})
		log.Printf("[INFO] AWS EC2 instance detected via default metadata" +
			" API endpoint, EC2RoleProvider added to the auth chain")
	} else {
		if usedEndpoint == "" {
			usedEndpoint = "default location"
		}
		log.Printf("[WARN] Ignoring AWS metadata API endpoint at %s "+
			"as it doesn't return any instance-id", usedEndpoint)
	}

	return awsCredentials.NewChainCredentials(providers)
}

func setOptionalEndpoint(cfg *aws.Config) string {
	endpoint := os.Getenv("AWS_METADATA_URL")
	if endpoint != "" {
		log.Printf("[INFO] Setting custom metadata endpoint: %q", endpoint)
		cfg.Endpoint = aws.String(endpoint)
		return endpoint
	}
	return ""
}
