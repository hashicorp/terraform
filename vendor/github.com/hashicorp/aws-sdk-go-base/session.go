package awsbase

import (
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/hashicorp/go-cleanhttp"
)

// GetSessionOptions attempts to return valid AWS Go SDK session authentication
// options based on pre-existing credential provider, configured profile, or
// fallback to automatically a determined session via the AWS Go SDK.
func GetSessionOptions(c *Config) (*session.Options, error) {
	options := &session.Options{
		Config: aws.Config{
			HTTPClient: cleanhttp.DefaultClient(),
			MaxRetries: aws.Int(0),
			Region:     aws.String(c.Region),
		},
	}

	creds, err := GetCredentials(c)
	if err != nil {
		return nil, err
	}

	// Call Get to check for credential provider. If nothing found, we'll get an
	// error, and we can present it nicely to the user
	cp, err := creds.Get()
	if err != nil {
		if IsAWSErr(err, "NoCredentialProviders", "") {
			// If a profile wasn't specified, the session may still be able to resolve credentials from shared config.
			if c.Profile == "" {
				sess, err := session.NewSession()
				if err != nil {
					return nil, errors.New(`No valid credential sources found for AWS Provider.
	Please see https://terraform.io/docs/providers/aws/index.html for more information on
	providing credentials for the AWS Provider`)
				}
				_, err = sess.Config.Credentials.Get()
				if err != nil {
					return nil, errors.New(`No valid credential sources found for AWS Provider.
	Please see https://terraform.io/docs/providers/aws/index.html for more information on
	providing credentials for the AWS Provider`)
				}
				log.Printf("[INFO] Using session-derived AWS Auth")
				options.Config.Credentials = sess.Config.Credentials
			} else {
				log.Printf("[INFO] AWS Auth using Profile: %q", c.Profile)
				options.Profile = c.Profile
				options.SharedConfigState = session.SharedConfigEnable
			}
		} else {
			return nil, fmt.Errorf("Error loading credentials for AWS Provider: %s", err)
		}
	} else {
		// add the validated credentials to the session options
		log.Printf("[INFO] AWS Auth provider used: %q", cp.ProviderName)
		options.Config.Credentials = creds
	}

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

// GetSession attempts to return valid AWS Go SDK session
func GetSession(c *Config) (*session.Session, error) {
	options, err := GetSessionOptions(c)

	if err != nil {
		return nil, err
	}

	sess, err := session.NewSessionWithOptions(*options)
	if err != nil {
		if IsAWSErr(err, "NoCredentialProviders", "") {
			return nil, errors.New(`No valid credential sources found for AWS Provider.
  Please see https://terraform.io/docs/providers/aws/index.html for more information on
  providing credentials for the AWS Provider`)
		}
		return nil, fmt.Errorf("Error creating AWS session: %s", err)
	}

	if c.MaxRetries > 0 {
		sess = sess.Copy(&aws.Config{MaxRetries: aws.Int(c.MaxRetries)})
	}

	for _, product := range c.UserAgentProducts {
		sess.Handlers.Build.PushBack(request.MakeAddToUserAgentHandler(product.Name, product.Version, product.Extra...))
	}

	// Generally, we want to configure a lower retry theshold for networking issues
	// as the session retry threshold is very high by default and can mask permanent
	// networking failures, such as a non-existent service endpoint.
	// MaxRetries will override this logic if it has a lower retry threshold.
	// NOTE: This logic can be fooled by other request errors raising the retry count
	//       before any networking error occurs
	sess.Handlers.Retry.PushBack(func(r *request.Request) {
		// We currently depend on the DefaultRetryer exponential backoff here.
		// ~10 retries gives a fair backoff of a few seconds.
		if r.RetryCount < 9 {
			return
		}
		// RequestError: send request failed
		// caused by: Post https://FQDN/: dial tcp: lookup FQDN: no such host
		if IsAWSErrExtended(r.Error, "RequestError", "send request failed", "no such host") {
			log.Printf("[WARN] Disabling retries after next request due to networking issue")
			r.Retryable = aws.Bool(false)
		}
		// RequestError: send request failed
		// caused by: Post https://FQDN/: dial tcp IPADDRESS:443: connect: connection refused
		if IsAWSErrExtended(r.Error, "RequestError", "send request failed", "connection refused") {
			log.Printf("[WARN] Disabling retries after next request due to networking issue")
			r.Retryable = aws.Bool(false)
		}
	})

	if !c.SkipCredsValidation {
		stsClient := sts.New(sess.Copy(&aws.Config{Endpoint: aws.String(c.StsEndpoint)}))
		if _, _, err := GetAccountIDAndPartitionFromSTSGetCallerIdentity(stsClient); err != nil {
			return nil, fmt.Errorf("error validating provider credentials: %s", err)
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

	iamClient := iam.New(sess.Copy(&aws.Config{Endpoint: aws.String(c.IamEndpoint)}))
	stsClient := sts.New(sess.Copy(&aws.Config{Endpoint: aws.String(c.StsEndpoint)}))

	if !c.SkipCredsValidation {
		accountID, partition, err := GetAccountIDAndPartitionFromSTSGetCallerIdentity(stsClient)

		if err != nil {
			return nil, "", "", fmt.Errorf("error validating provider credentials: %s", err)
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
				"Errors: %s", err)
	}

	var partition string
	if p, ok := endpoints.PartitionForRegion(endpoints.DefaultPartitions(), c.Region); ok {
		partition = p.ID()
	}

	return sess, "", partition, nil
}
