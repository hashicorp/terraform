## 1.0.1 (Unreleased)

* Updated the libraries that we share with Terraform core so that this provider can now use all the same backend features as Terraform Core v0.10.8.

## 1.0.0 (September 14, 2017)

ENHANCEMENTS:

* `terraform_remote_state` now accepts backend configuration arguments that were introduced to the backends in Terraform 0.10, including the `s3` backend's `workspace_dir_prefix` argument. ([#6](https://github.com/terraform-providers/terraform-provider-terraform/issues/6))
* New argument `defaults` on `terraform_remote_state` allows setting default values for outputs that are not set in the remote state. ([#11](https://github.com/terraform-providers/terraform-provider-terraform/issues/11))

## 0.1.0 (June 21, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
