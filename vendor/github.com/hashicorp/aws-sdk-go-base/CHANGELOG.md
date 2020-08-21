# v0.6.0 (unreleased)

BREAKING CHANGES

* AWS error checking function have been moved to `tfawserr` package. `IsAWSErr` has been renamed to `ErrMessageContains` and `IsAWSErrExtended` has been renamed to `ErrMessageAndOrigErrContain`. #37

ENHANCEMENTS

* Additional AWS error checking function have been added to the `tfawserr` package - `ErrCodeEquals`, `ErrCodeContains` and `ErrStatusCodeEquals`.

# v0.5.0 (June 4, 2020)

BREAKING CHANGES

* Credential ordering has changed from static, environment, shared credentials, EC2 metadata, default AWS Go SDK (shared configuration, web identity, ECS, EC2 Metadata) to static, environment, shared credentials, default AWS Go SDK (shared configuration, web identity, ECS, EC2 Metadata). #20
* The `AWS_METADATA_TIMEOUT` environment variable no longer has any effect as we now depend on the default AWS Go SDK EC2 Metadata client timeout of one second with two retries. #20 / #44

ENHANCEMENTS

* Always enable AWS shared configuration file support (no longer require `AWS_SDK_LOAD_CONFIG` environment variable) #38
* Automatically expand `~` prefix for home directories in shared credentials filename handling #40
* Support assume role duration, policy ARNs, tags, and transitive tag keys via configuration #39
* Add `CannotAssumeRoleError` and `NoValidCredentialSourcesError` error types with helpers #42

BUG FIXES

* Properly use custom STS endpoint during AssumeRole API calls triggered by Terraform AWS Provider and S3 Backend configurations #32
* Properly use custom EC2 metadata endpoint during API calls triggered by fallback credentials lookup #32
* Prefer shared configuration handling over EC2 metadata #20
* Prefer ECS credentials over EC2 metadata #20
* Remove hardcoded AWS Provider messaging in error messages #31 / #42

# v0.4.0 (October 3, 2019)

BUG FIXES

* awsauth: fixed credentials retrieval, validation, and error handling

# v0.3.0 (February 26, 2019)

BUG FIXES

* session: Return error instead of logging with AWS Account ID lookup failure [GH-3]

# v0.2.0 (February 20, 2019)

ENHANCEMENTS

* validation: Add `ValidateAccountID` and `ValidateRegion` functions [GH-1]

# v0.1.0 (February 18, 2019)

* Initial release after split from github.com/terraform-providers/terraform-provider-aws
