ephemeral_module "source" {
  source = "./source"
}

component "input" {
  source = "./consumer"
  inputs = {
    # Ephemeral value can't go here since "value" is not marked as ephemeral
    value = ephemeral_module.source.value
  }
}
