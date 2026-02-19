removed {
  from = module.foo.aws_instance.foo

  provisioner "shell" {
    when     = "destroy"

    // Capture that we can reference either count.index or each.key from a
    // removed block, and it's up to the user to ensure the provisioner is
    // correct for the now removed resources.
    command  = "destroy ${try(count.index, each.key)} ${self.foo}"
  }
}
