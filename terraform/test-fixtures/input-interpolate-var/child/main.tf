variable "length" { }

resource "template_file" "temp" {
  count     = var.length
  template  = "foo"
}
