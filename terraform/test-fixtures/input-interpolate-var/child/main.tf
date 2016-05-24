variable "list" { }
resource "template_file" "temp" {
  count     = "${length(split(",", var.list))}"
  template  = "foo"
}
