---
layout: "digitalocean"
page_title: "DigitalOcean: digitalocean_certificate"
sidebar_current: "docs-do-resource-certificate"
description: |-
  Provides a DigitalOcean Certificate resource.
---

# digitalocean\_certificate

Provides a DigitalOcean Certificate resource that allows you to manage
certificates for configuring TLS termination in Load Balancers.
Certificates created with this resource can be referenced in your
Load Balancer configuration via their ID.

## Example Usage

```hcl
# Create a new TLS certificate
resource "digitalocean_certificate" "cert" {
  name              = "Terraform Example"
  private_key       = "${file("/Users/terraform/certs/privkey.pem")}"
  leaf_certificate  = "${file("/Users/terraform/certs/cert.pem")}"
  certificate_chain = "${file("/Users/terraform/certs/fullchain.pem")}"
}

# Create a new Load Balancer with TLS termination
resource "digitalocean_loadbalancer" "public" {
  name        = "secure-loadbalancer-1"
  region      = "nyc3"
  droplet_tag = "backend"

  forwarding_rule {
    entry_port      = 443
    entry_protocol  = "https"

    target_port     = 80
    target_protocol = "http"

    certificate_id  = "${digitalocean_certificate.cert.id}"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the certificate for identification.
* `private_key` - (Required) The contents of a PEM-formatted private-key
corresponding to the SSL certificate.
* `leaf_certificate` - (Required) The contents of a PEM-formatted public
TLS certificate.
* `certificate_chain` - (Optional) The full PEM-formatted trust chain
between the certificate authority's certificate and your domain's TLS
certificate.

## Attributes Reference

The following attributes are exported:

* `id` - The unique ID of the certificate
* `name` - The name of the certificate
* `not_after` - The expiration date of the certificate
* `sha1_fingerprint` - The SHA-1 fingerprint of the certificate
