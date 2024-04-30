terraform {
  required_providers {
    boop = {
      source = "foobar/beep" # intentional mismatch between local name and type
    }
  }
  # TODO: Remove this if ephemeral values / resources get stabilized. If this
  # experiment is removed without stabilization, also remove the
  # "ephemeral" block below and the test cases it's supporting.
  experiments = [ephemeral_values]
}

resource "aws_instance" "no_count" {
}

resource "aws_instance" "count" {
  count = 1
}

resource "boop_instance" "yep" {
}

resource "boop_whatever" "nope" {
}

data "beep" "boop" {
}

ephemeral "beep" "boop" {
  provider = boop
}

check "foo" {
  data "boop_data" "boop_nested" {}

  assert {
    condition     = data.boop_data.boop_nested.id == null
    error_message = "check failed"
  }
}
