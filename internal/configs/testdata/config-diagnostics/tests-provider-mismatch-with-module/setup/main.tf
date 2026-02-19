terraform {
    required_providers {
        foo = {
            source = "hashicorp/bar"
            configuration_aliases = [ foo.bar ]
        }
        bar = {
            source = "hashicorp/foo"
        }
    }
}

resource "foo_resource" "resource" {}

resource "bar_resource" "resource" {}
