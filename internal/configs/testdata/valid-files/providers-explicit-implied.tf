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

import {
  provider = template.foo
  id       = "directory/foo_filename"
  to       = local_file.bar
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
