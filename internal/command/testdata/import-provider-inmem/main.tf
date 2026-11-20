terraform {
    backend "inmem" {
        force_write_error = true
    }
}

provider "test" {}

resource "test_instance" "foo" {
}
