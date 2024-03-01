removed {
  from = aws_instance.foo

  provisioner "shell" {
    when     = "destroy"
    command  = "destroy ${each.key} ${self.foo}"
  }
}
