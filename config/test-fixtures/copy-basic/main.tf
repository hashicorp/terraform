variable "ref" {
  default = "foo"
}

resource "foo" "bar" {
  depends_on = ["dep"]
  provider = "foo-west"
  count = 2
  attr  = "value"
  ref   = "${var.ref}"

  provisioner "shell" {
    inline = "echo"
  }

  lifecycle {
    ignore_changes = ["config"]
  }
}
