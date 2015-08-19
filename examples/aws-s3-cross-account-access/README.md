# S3 bucket with cross-account access

This example describes how to create an S3 bucket in one AWS account and give access to that bucket to another user from another AWS account using bucket policy.
It demonstrates capabilities of provider aliases.

See [more in the S3 documentation](http://docs.aws.amazon.com/AmazonS3/latest/dev/example-walkthroughs-managing-access-example2.html).

## How to run

Either `cp terraform.template.tfvars terraform.tfvars` and modify that new file accordingly or provide variables via CLI:

```
terraform apply \
	-var="prod_access_key=AAAAAAAAAAAAAAAAAAA" \
	-var="prod_secret_key=SuperSecretKeyForAccountA" \
	-var="test_account_id=123456789012" \
	-var="test_access_key=BBBBBBBBBBBBBBBBBBB" \
	-var="test_secret_key=SuperSecretKeyForAccountB" \
	-var="bucket_name=tf-bucket-in-prod" \
```
