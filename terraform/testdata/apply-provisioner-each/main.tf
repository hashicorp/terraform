resource "aws_instance" "bar" {
    for_each = toset(["a"])
    provisioner "shell" {
      when = "destroy"
      command = "echo ${each.value}"
    }
}
