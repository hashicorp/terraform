---
layout: "aws"
page_title: "AWS: aws_iam_server_certificate"
sidebar_current: "docs-aws-resource-iam-server-certificate"
description: |-
  Provides an IAM Server Certificate
---

# aws\_iam\_server\_certificate

Provides an IAM Server Certificate resource to upload Server Certificates.
Certs uploaded to IAM can easily work with other AWS services such as:

- AWS Elastic Beanstalk
- Elastic Load Balancing
- CloudFront
- AWS OpsWorks

For information about server certificates in IAM, see [Managing Server
Certficates][2] in AWS Documentation.

## Example Usage

**Using certs on file:**

```
resource "aws_iam_server_certificate" "test_cert" {
  name = "some_test_cert"
  certificate_body = "${file("self-ca-cert.pem")}"
  private_key = "${file("test-key.pem")}"
}
```

**Example with cert in-line:**

```
resource "aws_iam_server_certificate" "test_cert_alt" {
  name = "alt_test_cert"
  certificate_body = <<EOF
-----BEGIN CERTIFICATE-----
[......] # cert contents
-----END CERTIFICATE-----
EOF

  private_key =  <<EOF
-----BEGIN RSA PRIVATE KEY-----
[......] # cert contents
-----END CERTIFICATE-----
EOF
}
```

**Use in combination with an AWS ELB resource:**

```
resource "aws_iam_server_certificate" "test_cert" {
  name = "some_test_cert"
  certificate_body = "${file("self-ca-cert.pem")}"
  private_key = "${file("test-key.pem")}"
}

resource "aws_elb" "ourapp" {
  name = "terraform-asg-deployment-example"
  availability_zones = ["us-west-2a"]
  cross_zone_load_balancing = true

  listener {
    instance_port = 8000
    instance_protocol = "http"
    lb_port = 443
    lb_protocol = "https"
    ssl_certificate_id = "${aws_iam_server_certificate.test_cert.arn}"
  }
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) The name of the Server Certificate. Do not include the 
  path in this value.
* `certificate_body` – (Required) The contents of the public key certificate in 
  PEM-encoded format.
* `certificate_chain` – (Optional) The contents of the certificate chain. 
  This is typically a concatenation of the PEM-encoded public key certificates 
  of the chain. 
* `private_key` – (Required) The contents of the private key in PEM-encoded format.
* `path` - (Optional) The IAM path for the server certificate.  If it is not 
    included, it defaults to a slash (/). If this certificate is for use with
    AWS CloudFront, the path must be in format `/cloudfront/your_path_here`.
    See [IAM Identifiers][1] for more details on IAM Paths.

~> **NOTE:** AWS performs behind-the-scenes modifications to some certificate files if they do not adhere to a specific format. These modifications will result in terraform forever believing that it needs to update the resources since the local and AWS file contents will not match after theses modifications occur. In order to prevent this from happening you must ensure that all your PEM-encoded files use UNIX line-breaks and that `certificate_body` contains only one certificate. All other certificates should go in `certificate_chain`. It is common for some Certificate Authorities to issue certificate files that have DOS line-breaks and that are actually multiple certificates concatenated together in order to form a full certificate chain.

## Attributes Reference

* `id` - The unique Server Certificate name
* `name` - The name of the Server Certificate
* `arn` - The Amazon Resource Name (ARN) specifying the server certificate.


[1]: http://docs.aws.amazon.com/IAM/latest/UserGuide/Using_Identifiers.html
[2]: http://docs.aws.amazon.com/IAM/latest/UserGuide/ManagingServerCerts.html
