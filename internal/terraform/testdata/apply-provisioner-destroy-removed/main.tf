terraform {
  # provisioners in removed blocks are currently only experimental
  experiments = [removed_provisioners]
}

removed {
  from = aws_instance.foo

  provisioner "shell" {
    when     = "destroy"
    command  = "destroy ${each.key} ${self.foo}"
  }
}
