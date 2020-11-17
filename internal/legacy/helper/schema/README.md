# Terraform Helper Lib: schema

The `schema` package provides a high-level interface for writing resource
providers for Terraform.

If you're writing a resource provider, we recommend you use this package.

The interface exposed by this package is much friendlier than trying to
write to the Terraform API directly. The core Terraform API is low-level
and built for maximum flexibility and control, whereas this library is built
as a framework around that to more easily write common providers.
