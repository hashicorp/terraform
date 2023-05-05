provider "aws" {

}

provider "template" {
  alias = "foo"
}

resource "aws_instance" "foo" {

}

resource "null_resource" "foo" {

}

import {
  id = "directory/filename"
  to = local_file.foo
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
