variable "c" { default = 1 }

resource "template_file" "parent" {
  count = "${var.c}"
  template = "Hi"
}

resource "template_file" "child" {
  template = "${join(",", template_file.parent.*.template)} ok"
}
