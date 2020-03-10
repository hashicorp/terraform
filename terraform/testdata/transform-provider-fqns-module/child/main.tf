terraform {
  required_providers {
    your_aws = {
      // This is temporarily using the legacy provider namespace so that we can
      // write tests without fully supporting provider source
      source = "-/aws"
    }
  }
}

resource "aws_instance" "web" {
  provider = "your_aws"
}
