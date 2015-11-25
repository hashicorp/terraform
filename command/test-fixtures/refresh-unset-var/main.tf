variable "should_ask" {}

provider "test" {}

resource "test_instance" "foo" {
  ami = "${var.should_ask}"
}
