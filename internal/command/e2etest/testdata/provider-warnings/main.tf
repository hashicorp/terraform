terraform {
    required_providers {
        terraform = {
            // hashicorp/terraform is published in the registry, but it is
            // archived (since it is internal) and returns a warning:
            //
            // "This provider is archived and no longer needed. The terraform_remote_state
            // data source is built into the latest Terraform release."
            source = "hashicorp/terraform"
        }
    }
}
