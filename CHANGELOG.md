## 1.0.0 (Unreleased)

ENHANCEMENTS:

* `terraform_remote_state` now accepts backend configuration arguments that were introduced to the backends in Terraform 0.10, including the `s3` backend's `workspace_dir_prefix` argument. [GH-6]
* New argument `defaults` on `terraform_remote_state` allows setting default values for outputs that are not set in the remote state. [GH-11]

## 0.1.0 (June 21, 2017)

NOTES:

* Same functionality as that of Terraform 0.9.8. Repacked as part of [Provider Splitout](https://www.hashicorp.com/blog/upcoming-provider-changes-in-terraform-0-10/)
