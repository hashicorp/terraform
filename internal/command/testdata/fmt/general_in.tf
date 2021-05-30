# This test case is intended to cover many of the main formatting
# rules of "terraform fmt" at once. It's fine to add new stuff in
# here, but you can also add other _in.tf/_out.tf pairs in the
# same directory if you want to test something complicated that,
# for example, requires specific nested context.
#
# The input file of this test intentionally has strange whitespace
# alignment, because the goal is to see the fmt command fix it.
# If you're applying batch formatting to all .tf files in the
# repository (or similar), be sure to skip this one to avoid
# invalidating the test.

terraform {
required_providers {
foo = { version = "1.0.0" }
barbaz = {
            version = "2.0.0"
}
}
}

variable instance_type {

}

resource foo_instance foo {
  instance_type = "${var.instance_type}"
}

resource foo_instance "bar" {
    instance_type = "${var.instance_type}-2"
}

resource "foo_instance" /* ... */ "baz" {
  instance_type = "${var.instance_type}${var.instance_type}"

  beep boop {}
  beep blep {
    thingy = "${var.instance_type}"
  }
}

  provider "" {
}

locals {
  name = "${contains(["foo"], var.my_var) ? "${var.my_var}-bar" :
    contains(["baz"], var.my_var) ? "baz-${var.my_var}" :
  file("ERROR: unsupported type ${var.my_var}")}"
  wrapped = "${(var.my_var == null ? 1 :
    var.your_var == null ? 2 :
  3)}"
}
