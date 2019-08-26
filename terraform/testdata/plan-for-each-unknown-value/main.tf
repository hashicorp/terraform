# expressions with variable reference
variable "foo" {
  type = string
}

resource "aws_instance" "foo" {
  for_each = toset(
       [for i in range(0,3) : sha1("${i}${var.foo}")]
    )
  foo = "foo"
}
