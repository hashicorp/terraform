resource "test_instance" "tmpl" {
  foo = "${file("${path.module}/template.txt")}"
}

output "template" {
	value = "${test_instance.tmpl.foo}"
}

terraform {
  required_providers {
    test = {
      source = "hashicorp/test"
    }
  }
}
