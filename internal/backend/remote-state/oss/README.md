# OSS backend: PrivateLink endpoints

Set `force_path_style = true` when using an OSS PrivateLink endpoint:

```hcl
terraform {
  backend "oss" {
    bucket           = "bucket-for-terraform-state"
    prefix           = "path/mystate"
    key              = "version-1.tfstate"
    region           = "ap-southeast-5"
    endpoint         = "https://ep-example.oss.ap-southeast-5.privatelink.aliyuncs.com"
    force_path_style = true
  }
}
```

`force_path_style` is optional and defaults to `false`. When enabled, OSS
requests use `https://<endpoint>/<bucket>/<object>` instead of
`https://<bucket>.<endpoint>/<object>`. This lets the backend use PrivateLink
endpoints whose DNS names do not support bucket subdomains. Supply your own
endpoint and bucket, and configure credentials separately.

This option affects OSS requests. Configure TableStore separately if you use it
for state locking and consistency checks.
