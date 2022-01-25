---
name: Bug report
about: Let us know about an unexpected error, a crash, or an incorrect behavior.
labels: bug, new

---

<!--
Hi there,

Thank you for opening an issue. Please note that we try to keep the Terraform issue tracker reserved for bug reports and feature requests. For general usage questions, please see: https://www.terraform.io/community.html.

If your issue relates to Terraform Cloud/Enterprise, please contact tf-cloud@hashicorp.support.
If your issue relates to a specific Terraform provider, please open it in the provider's own repository. The index of providers is at https://registry.terraform.io/browse/providers.

To fix problems, we need clear reproduction cases - we need to be able to see it happen locally. A reproduction case is ideally something a Terraform Core engineer can git-clone or copy-paste and run immediately, without inventing any details or context. 

* A short example can be directly copy-pasteable; longer examples should be in separate git repositories, especially if multiple files are needed
* Please include all needed context. For example, if you figured out that an expression can cause a crash, put the expression in a variable definition or a resource
* Set defaults on (or omit) any variables. The person reproducing it should not need to invent variable settings
* If multiple steps are required, such as running terraform twice, consider scripting it in a simple shell script. Providing a script can be easier than explaining what changes to make to the config between runs.
* Omit any unneeded complexity: remove variables, conditional statements, functions, modules, providers, and resources that are not needed to trigger the bug
* When possible, use the [null resource](https://www.terraform.io/docs/providers/null/resource.html) provider rather than a real provider in order to minimize external dependencies. We know this isn't always feasible. The Terraform Core team doesn't have deep domain knowledge in every provider, or access to every cloud platform for reproduction cases.

-->

### Terraform Version
<!---
Run `terraform version` to show the version, and paste the result between the ``` marks below.

If you are not running the latest version of Terraform, please try upgrading because your issue may have already been fixed.
-->

```
...
```

### Terraform Configuration Files
<!--
Paste the relevant parts of your Terraform configuration between the ``` marks below.

For Terraform configs larger than a few resources, or that involve multiple files, please make a GitHub repository that we can clone, rather than copy-pasting multiple files in here. For security, you can also encrypt the files using our GPG public key at https://www.hashicorp.com/security.
-->

```terraform
...
```

### Debug Output
<!--
Full debug output can be obtained by running Terraform with the environment variable `TF_LOG=trace`. Please create a GitHub Gist containing the debug output. Please do _not_ paste the debug output in the issue, since debug output is long.

Debug output may contain sensitive information. Please review it before posting publicly, and if you are concerned feel free to encrypt the files using the HashiCorp security public key.
-->

### Expected Behavior
<!--
What should have happened?
-->

### Actual Behavior
<!--
What actually happened?
-->

### Steps to Reproduce
<!--
Please list the full steps required to reproduce the issue, for example:
1. `terraform init`
2. `terraform apply`
-->

### Additional Context
<!--
Are there anything atypical about your situation that we should know? For example: is Terraform running in a wrapper script or in a CI system? Are you passing any unusual command line options or environment variables to opt-in to non-default behavior?
-->

### References
<!--
Are there any other GitHub issues (open or closed) or Pull Requests that should be linked here? For example:

- #6017

-->
