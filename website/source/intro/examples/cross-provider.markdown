---
layout: "intro"
page_title: "Cross Provider"
sidebar_current: "examples-cross-provider"
---

# Cross Provider Example

This is a simple example of the cross-provider capabilities of
Terraform.

Very simply, this creates a Heroku application and points a DNS
CNAME record at the result via DNSimple. A `host` query to the outputted
hostname should reveal the correct DNS configuration.

## Command

```
terraform apply \
    -var 'heroku_email=YOUR_EMAIL' \
    -var 'heroku_api_key=YOUR_KEY' \
    -var 'dnsimple_domain=example.com' \
    -var 'dnsimple_email=YOUR_EMAIL' \
    -var 'dnsimple_token=YOUR_TOKEN'
```

## Configuration

```
variable "heroku_email" {}
variable "heroku_api_key" {}

# The domain we are creating a record for
variable "dnsimple_domain" {}

variable "dnsimple_token" {}
variable "dnsimple_email" {}


# Specify the provider and access details
provider "heroku" {
    email = "${var.heroku_email}"
    api_key = "${var.heroku_api_key}"
}

# Create our Heroku application. Heroku will
# automatically assign a name.
resource "heroku_app" "web" {
}

# Create our DNSimple record to point to the
# heroku application.
resource "dnsimple_record" "web" {
  domain = "${var.dnsimple_domain}"

  name = "terraform"

  # heroku_hostname is a computed attribute on the heroku
  # application we can use to determine the hostname
  value = "${heroku_app.web.heroku_hostname}"

  type = "CNAME"
  ttl = 3600
}

# The Heroku domain, which will be created and added
# to the heroku application after we have assigned the domain
# in DNSimple
resource "heroku_domain" "foobar" {
    app = "${heroku_app.web.name}"
    hostname = "${dnsimple_record.web.hostname}"
}

# Output the hostname of the newly created record
output "address" {
  value = "${dnsimple_record.web.hostname}"
}
```
