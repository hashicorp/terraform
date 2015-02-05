## How to test

### S3 remote state storage
To run S3 integration tests you need following env variables to be set:
  * AWS_ACCESS_KEY
  * AWS_SECRET_KEY
  * AWS_DEFAULT_REGION
  * TERRAFORM_STATE_BUCKET

Additionally specified bucket should exist in the defined region and should be accessible 
using specified credentials.
