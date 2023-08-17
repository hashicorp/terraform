variable "foo" { default = "bar" }

mnptu {
    backend "local" {
        path = "${var.foo}"
    }
}
