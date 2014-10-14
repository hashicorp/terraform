# Cross Provider Examples

This is a simple example of the cross-provider capabilities of
Terraform.

Very simply, this creates a Heroku application and points a DNS
CNAME record at the result via DNSimple. A `host` query to the outputted
hostname should reveal the correct DNS configuration.

As with all examples, just copy and paste the example and run
`terraform apply` to see it work.
