terraform {
    backend "inmem" {}
}

output "foo" {
    value = "bar"
}
