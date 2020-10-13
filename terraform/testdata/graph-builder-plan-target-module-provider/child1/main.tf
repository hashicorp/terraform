variable "key" {}

provider "test" {
  test_string = "${var.key}"
}

resource "test_object" "foo" {}
