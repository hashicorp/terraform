variable "num" {
  default = 3
}

resource "aws_instance" "a" {
  count = var.num
}

resource "aws_instance" "b" {
  provisioner "local-exec" {
    # Since we're in a provisioner block here, this expression is
    # resolved during the apply walk and so the resource count must
    # be known during that walk, even though apply walk doesn't
    # do DynamicExpand.
    command = "echo ${length(aws_instance.a)}"
  }
}
