package awsbase

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/hashicorp/aws-sdk-go-base/tfawserr"
	"github.com/hashicorp/go-cleanhttp"
)

const (
	// AppendUserAgentEnvVar is a conventionally used environment variable
	// containing additional HTTP User-Agent information.
	// If present and its value is non-empty, it is directly appended to the
	// User-Agent header for HTTP requests.
	AppendUserAgentEnvVar = "TF_APPEND_USER_AGENT"
	// Maximum network retries.
	// We depend on the AWS Go SDK DefaultRetryer exponential backoff.
	// Ensure that if the AWS Config MaxRetries is set high (which it is by
	// default), that we only retry for a few seconds with typically
	// unrecoverable network errors, such as DNS lookup failures.
	MaxNetworkRetryCount = 9
)

// GetSessionOptions attempts to return valid AWS Go SDK session authentication
// options based on pre-existing credential provider, configured profile, or
// fallback to automatically a determined session via the AWS Go SDK.
func GetSessionOptions(c *Config) (*session.Options, error) {
	options := &session.Options{
		Config: aws.Config{
			EndpointResolver: c.EndpointResolver(),
			HTTPClient:       cleanhttp.DefaultClient(),
			MaxRetries:       aws.Int(0),
			Region:           aws.String(c.Region),
		},
		Profile:           c.Profile,
		SharedConfigState: session.SharedConfigEnable,
	}

	// get and validate credentials
	creds, err := GetCredentials(c)
	if err != nil {
		return nil, err
	}

	// add the validated credentials to the session options
	options.Config.Credentials = creds

	if c.Insecure {
		transport := options.Config.HTTPClient.Transport.(*http.Transport)
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	if c.DebugLogging {
		options.Config.LogLevel = aws.LogLevel(aws.LogDebugWithHTTPBody | aws.LogDebugWithRequestRetries | aws.LogDebugWithRequestErrors)
		options.Config.Logger = DebugLogger{}
	}

	return options, nil
}

// GetSession attempts to return valid AWS Go SDK session.
func GetSession(c *Config) (*session.Session, error) {
	if c.SkipMetadataApiCheck {
		os.Setenv("AWS_EC2_METADATA_DISABLED", "true")
	}

	options, err := GetSessionOptions(c)

	if err != nil {
		return nil, err
	}

	sess, err := session.NewSessionWithOptions(*options)
	if err != nil {
		if tfawserr.ErrCodeEquals(err, "NoCredentialProviders") {
			return nil, c.NewNoValidCredentialSourcesError(err)
		}
		return nil, fmt.Errorf("Error creating AWS session: %w", err)
	}

	if c.MaxRetries > 0 {
		sess = sess.Copy(&aws.Config{MaxRetries: aws.Int(c.MaxRetries)})
	}

	for _, product := range c.UserAgentProducts {
		sess.Handlers.Build.PushBack(request.MakeAddToUserAgentHandler(product.Name, product.Version, product.Extra...))
	}

	// Add custom input from ENV to the User-Agent request header
	// Reference: https://github.com/terraform-providers/terraform-provider-aws/issues/9149
	if v := os.Getenv(AppendUserAgentEnvVar); v != "" {
		log.Printf("[DEBUG] Using additional User-Agent Info: %s", v)
		sess.Handlers.Build.PushBack(request.MakeAddToUserAgentFreeFormHandler(v))
	}

	// Generally, we want to configure a lower retry theshold for networking issues
	// as the session retry threshold is very high by default and can mask permanent
	// networking failures, such as a non-existent service endpoint.
	// MaxRetries will override this logic if it has a lower retry threshold.
	// NOTE: This logic can be fooled by other request errors raising the retry count
	//       before any networking error occurs
	sess.Handlers.Retry.PushBack(func(r *request.Request) {
		if r.RetryCount < MaxNetworkRetryCount {
			return
		}
		// RequestError: send request failed
		// caused by: Post https://FQDN/: dial tcp: lookup FQDN: no such host
		if tfawserr.ErrMessageAndOrigErrContain(r.Error, "RequestError", "send request failed", "no such host") {
			log.Printf("[WARN] Disabling retries after next request due to networking issue")
			r.Retryable = aws.Bool(false)
		}
		// RequestError: send request failed
		// caused by: Post https://FQDN/: dial tcp IPADDRESS:443: connect: connection refused
		if tfawserr.ErrMessageAndOrigErrContain(r.Error, "RequestError", "send request failed", "connection refused") {
			log.Printf("[WARN] Disabling retries after next request due to networking issue")
			r.Retryable = aws.Bool(false)
		}
	})

	if !c.SkipCredsValidation {
		if _, _, err := GetAccountIDAndPartitionFromSTSGetCallerIdentity(sts.New(sess)); err != nil {
			return nil, fmt.Errorf("error validating provider credentials: %w", err)
		}
	}

	return sess, nil
}

// GetSessionWithAccountIDAndPartition attempts to return valid AWS Go SDK session
// along with account ID and partition information if available
func GetSessionWithAccountIDAndPartition(c *Config) (*session.Session, string, string, error) {
	sess, err := GetSession(c)

	if err != nil {
		return nil, "", "", err
	}

	if c.AssumeRoleARN != "" {
		accountID, partition, _ := parseAccountIDAndPartitionFromARN(c.AssumeRoleARN)
		return sess, accountID, partition, nil
	}

	iamClient := iam.New(sess)
	stsClient := sts.New(sess)

	if !c.SkipCredsValidation {
		accountID, partition, err := GetAccountIDAndPartitionFromSTSGetCallerIdentity(stsClient)

		if err != nil {
			return nil, "", "", fmt.Errorf("error validating provider credentials: %w", err)
		}

		return sess, accountID, partition, nil
	}

	if !c.SkipRequestingAccountId {
		credentialsProviderName := ""

		if credentialsValue, err := sess.Config.Credentials.Get(); err == nil {
			credentialsProviderName = credentialsValue.ProviderName
		}

		accountID, partition, err := GetAccountIDAndPartition(iamClient, stsClient, credentialsProviderName)

		if err == nil {
			return sess, accountID, partition, nil
		}

		return nil, "", "", fmt.Errorf(
			"AWS account ID not previously found and failed retrieving via all available methods. "+
				"See https://www.terraform.io/docs/providers/aws/index.html#skip_requesting_account_id for workaround and implications. "+
				"Errors: %w", err)
	}

	var partition string
	if p, ok := endpoints.PartitionForRegion(endpoints.DefaultPartitions(), c.Region); ok {
		partition = p.ID()
	}

	return sess, "", partition, nil
}
