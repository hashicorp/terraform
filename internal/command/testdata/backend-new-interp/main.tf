variable "foo" { default = "bar" }

terraform {
    backend "local" {
        path = "${var.foo}"
    }
}
