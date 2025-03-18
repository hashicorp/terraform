component "vault-config" {
  source = "./"
  inputs = {
    ssh_key_private = component.boundary.ssh_key_private
  }
}

component "boundary" {
  source = "./"
  inputs = {
    boundary_vault_token = component.vault-config.boundary_vault_token
  }
}
