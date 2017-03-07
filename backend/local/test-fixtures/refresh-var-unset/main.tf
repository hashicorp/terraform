variable "should_ask" {}

provider "test" {
  value = "${var.should_ask}"
}

resource "test_instance" "foo" {
  foo = "bar"
}
