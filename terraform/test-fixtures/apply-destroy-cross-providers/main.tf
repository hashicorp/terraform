resource "terraform_remote_state" "shared" {}

module "child" {
    source = "./child"
    value = "${terraform_remote_state.shared.output.env_name}"
}
