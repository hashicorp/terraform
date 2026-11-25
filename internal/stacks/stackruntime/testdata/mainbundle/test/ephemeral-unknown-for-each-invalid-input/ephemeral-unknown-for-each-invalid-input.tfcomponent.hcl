variable "names" {
  type = set(string)
}

ephemeral_module "main" {
  for_each = var.names

  source = "./"

  inputs = {
    # Only the ephemeral module's own evaluation can find the error in its module,
    # as the module's input is known but the validate walk treats all inputs as unknown.
    number = "not-a-number"
  }
}
