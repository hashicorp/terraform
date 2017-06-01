package aws

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	awsCredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-cleanhttp"
)

func GetAccountInfo(iamconn *iam.IAM, stsconn *sts.STS, authProviderName string) (string, string, error) {
	// If we have creds from instance profile, we can use metadata API
	if authProviderName == ec2rolecreds.ProviderName {
		log.Println("[DEBUG] Trying to get account ID via AWS Metadata API")

		cfg := &aws.Config{}
		setOptionalEndpoint(cfg)
		sess, err := session.NewSession(cfg)
		if err != nil {
			return "", "", errwrap.Wrapf("Error creating AWS session: {{err}}", err)
		}

		metadataClient := ec2metadata.New(sess)
		info, err := metadataClient.IAMInfo()
		if err != nil {
			// This can be triggered when no IAM Role is assigned
			// or AWS just happens to return invalid response
			return "", "", fmt.Errorf("Failed getting EC2 IAM info: %s", err)
		}

		return parseAccountInfoFromArn(info.InstanceProfileArn)
	}

	// Then try IAM GetUser
	log.Println("[DEBUG] Trying to get account ID via iam:GetUser")
	outUser, err := iamconn.GetUser(nil)
	if err == nil {
		return parseAccountInfoFromArn(*outUser.User.Arn)
	}

	awsErr, ok := err.(awserr.Error)
	// AccessDenied and ValidationError can be raised
	// if credentials belong to federated profile, so we ignore these
	if !ok || (awsErr.Code() != "AccessDenied" && awsErr.Code() != "ValidationError" && awsErr.Code() != "InvalidClientTokenId") {
		return "", "", fmt.Errorf("Failed getting account ID via 'iam:GetUser': %s", err)
	}
	log.Printf("[DEBUG] Getting account ID via iam:GetUser failed: %s", err)

	// Then try STS GetCallerIdentity
	log.Println("[DEBUG] Trying to get account ID via sts:GetCallerIdentity")
	outCallerIdentity, err := stsconn.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err == nil {
		return parseAccountInfoFromArn(*outCallerIdentity.Arn)
	}
	log.Printf("[DEBUG] Getting account ID via sts:GetCallerIdentity failed: %s", err)

	// Then try IAM ListRoles
	log.Println("[DEBUG] Trying to get account ID via iam:ListRoles")
	outRoles, err := iamconn.ListRoles(&iam.ListRolesInput{
		MaxItems: aws.Int64(int64(1)),
	})
	if err != nil {
		return "", "", fmt.Errorf("Failed getting account ID via 'iam:ListRoles': %s", err)
	}

	if len(outRoles.Roles) < 1 {
		return "", "", errors.New("Failed getting account ID via 'iam:ListRoles': No roles available")
	}

	return parseAccountInfoFromArn(*outRoles.Roles[0].Arn)
}

func parseAccountInfoFromArn(arn string) (string, string, error) {
	parts := strings.Split(arn, ":")
	if len(parts) < 5 {
		return "", "", fmt.Errorf("Unable to parse ID from invalid ARN: %q", arn)
	}
	return parts[1], parts[4], nil
}

// This function is responsible for reading credentials from the
// environment in the case that they're not explicitly specified
// in the Terraform configuration.
func GetCredentials(c *Config) (*awsCredentials.Credentials, error) {
	// build a chain provider, lazy-evaulated by aws-sdk
	providers := []awsCredentials.Provider{
		&awsCredentials.StaticProvider{Value: awsCredentials.Value{
			AccessKeyID:     c.AccessKey,
			SecretAccessKey: c.SecretKey,
			SessionToken:    c.Token,
		}},
		&awsCredentials.EnvProvider{},
		&awsCredentials.SharedCredentialsProvider{
			Filename: c.CredsFilename,
			Profile:  c.Profile,
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

	if !c.SkipMetadataApiCheck {
		// Real AWS should reply to a simple metadata request.
		// We check it actually does to ensure something else didn't just
		// happen to be listening on the same IP:Port
		metadataClient := ec2metadata.New(session.New(cfg))
		if metadataClient.Available() {
			providers = append(providers, &ec2rolecreds.EC2RoleProvider{
				Client: metadataClient,
			})
			log.Print("[INFO] AWS EC2 instance detected via default metadata" +
				" API endpoint, EC2RoleProvider added to the auth chain")
		} else {
			if usedEndpoint == "" {
				usedEndpoint = "default location"
			}
			log.Printf("[INFO] Ignoring AWS metadata API endpoint at %s "+
				"as it doesn't return any instance-id", usedEndpoint)
		}
	}

	// This is the "normal" flow (i.e. not assuming a role)
	if c.AssumeRoleARN == "" {
		return awsCredentials.NewChainCredentials(providers), nil
	}

	// Otherwise we need to construct and STS client with the main credentials, and verify
	// that we can assume the defined role.
	log.Printf("[INFO] Attempting to AssumeRole %s (SessionName: %q, ExternalId: %q, Policy: %q)",
		c.AssumeRoleARN, c.AssumeRoleSessionName, c.AssumeRoleExternalID, c.AssumeRolePolicy)

	creds := awsCredentials.NewChainCredentials(providers)
	cp, err := creds.Get()
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoCredentialProviders" {
			return nil, errors.New(`No valid credential sources found for AWS Provider.
  Please see https://terraform.io/docs/providers/aws/index.html for more information on
  providing credentials for the AWS Provider`)
		}

		return nil, fmt.Errorf("Error loading credentials for AWS Provider: %s", err)
	}

	log.Printf("[INFO] AWS Auth provider used: %q", cp.ProviderName)

	awsConfig := &aws.Config{
		Credentials:      creds,
		Region:           aws.String(c.Region),
		MaxRetries:       aws.Int(c.MaxRetries),
		HTTPClient:       cleanhttp.DefaultClient(),
		S3ForcePathStyle: aws.Bool(c.S3ForcePathStyle),
	}

	stsclient := sts.New(session.New(awsConfig))
	assumeRoleProvider := &stscreds.AssumeRoleProvider{
		Client:  stsclient,
		RoleARN: c.AssumeRoleARN,
	}
	if c.AssumeRoleSessionName != "" {
		assumeRoleProvider.RoleSessionName = c.AssumeRoleSessionName
	}
	if c.AssumeRoleExternalID != "" {
		assumeRoleProvider.ExternalID = aws.String(c.AssumeRoleExternalID)
	}
	if c.AssumeRolePolicy != "" {
		assumeRoleProvider.Policy = aws.String(c.AssumeRolePolicy)
	}

	providers = []awsCredentials.Provider{assumeRoleProvider}

	assumeRoleCreds := awsCredentials.NewChainCredentials(providers)
	_, err = assumeRoleCreds.Get()
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok && awsErr.Code() == "NoCredentialProviders" {
			return nil, fmt.Errorf("The role %q cannot be assumed.\n\n"+
				"  There are a number of possible causes of this - the most common are:\n"+
				"    * The credentials used in order to assume the role are invalid\n"+
				"    * The credentials do not have appropriate permission to assume the role\n"+
				"    * The role ARN is not valid",
				c.AssumeRoleARN)
		}

		return nil, fmt.Errorf("Error loading credentials for AWS Provider: %s", err)
	}

	return assumeRoleCreds, nil
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
