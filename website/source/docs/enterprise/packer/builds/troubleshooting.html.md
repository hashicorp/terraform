---
layout: "enterprise"
page_title: "Troubleshooting - Packer Builds - Terraform Enterprise"
sidebar_current: "docs-enterprise-packerbuilds-troubleshooting"
description: |-
  Packer builds can fail in Terraform Enterprise for a number of reasons – improper configuration, transient networking errors, and hardware constraints are all possible.
---

# Troubleshooting Failing Builds

Packer builds can fail in Terraform Enterprise for a number of reasons –
improper configuration, transient networking errors, and hardware constraints
are all possible. Below is a list of debugging options you can use.

### Verbose Packer Logging

You can [set a variable](/docs/enterprise/packer/builds/build-environment.html#environment-variables) in the UI that increases the logging verbosity
in Packer. Set the `PACKER_LOG` key to a value of `1` to accomplish this.

After setting the variable, you'll need to [rebuild](/docs/enterprise/packer/builds/rebuilding.html).

Verbose logging will be much louder than normal Packer logs and isn't
recommended for day-to-day operations. Once enabled, you'll be able to see in
further detail why things failed or what operations Packer was performing.

This can also be used locally:

```text
$ PACKER_LOG=1 packer build ...
```

### Hanging Builds

Some VM builds, such as VMware or VirtualBox, may hang at various stages,
most notably `Waiting for SSH...`.

Things to pay attention to when this happens:

- SSH credentials must be properly configured. AWS keypairs should match, SSH
  usernames should be correct, passwords should match, etc.

- Any VM pre-seed configuration should have the same SSH configuration as your
  template defines

A good way to debug this is to manually attempt to use the same SSH
configuration locally, running with `packer build -debug`. See
more about [debugging Packer builds](https://packer.io/docs/other/debugging.html).

### Hardware Limitations

Your build may be failing by requesting larger memory or
disk usage then is available. Read more about the [build environment](/docs/enterprise/packer/builds/build-environment.html#hardware-limitations).

_Typically_ Packer builds that fail due to requesting hardware limits
that exceed Terraform Enterprise's [hardware limitations](/docs/enterprise/packer/builds/build-environment.html#hardware-limitations)
will fail with a _The operation was canceled_ error message as shown below:

```text
# ...
==> vmware-iso: Starting virtual machine...
    vmware-iso: The VM will be run headless, without a GUI. If you want to
    vmware-iso: view the screen of the VM, connect via VNC without a password to
    vmware-iso: 127.0.0.1:5918
==> vmware-iso: Error starting VM: VMware error: Error: The operation was canceled
==> vmware-iso: Waiting 4.604392397s to give VMware time to clean up...
==> vmware-iso: Deleting output directory...
Build 'vmware-iso' errored: Error starting VM: VMware error: Error: The operation was canceled

==> Some builds didn't complete successfully and had errors:
--> vmware-iso: Error starting VM: VMware error: Error: The operation was canceled
```

### Local Debugging

Sometimes it's faster to debug failing builds locally. In this case,
you'll want to [install Packer](https://www.packer.io/intro/getting-started/setup.html) and any providers (like Virtualbox) necessary.

Because Terraform Enterprise runs the open source version of Packer, there
should be no difference in execution between the two, other than the environment
that Packer is running in. For more on hardware constraints in the Terraform
Enterprise environment read below.

Once your builds are running smoothly locally you can push it up to Terraform
Enterprise for versioning and automated builds.

### Internal Errors

This is a short list of internal errors and what they mean.

- SIC-001: Your data was being ingressed from GitHub but failed
to properly unpack. This can be caused by bad permissions, using
symlinks or very large repository sizes. Using symlinks inside of the
packer directory, or the root of the repository, if the packer directory
is unspecified, will result in this internal error.

    _**Note:** Most often this error occurs when applications or builds are
    linked to a GitHub repository and the directory and/or template paths are
    incorrect. Double check that the paths specified when you linked the GitHub
    repository match the actual paths to your template file._

- SEC-001: Your data was being unpacked from a tarball uploaded
and encountered an error. This can be caused by bad permissions, using
symlinks or very large tarball sizes.

### Community Resources

Packer is an open source project with an active community. If you're
having an issue specific to Packer, the best avenue for support is
the mailing list or IRC. All bug reports should go to GitHub.

- Website: [packer.io](https://packer.io)
- GitHub: [github.com/mitchellh/packer](https://github.com/mitchellh/packer)
- IRC: `#packer-tool` on Freenode
- Mailing list: [Google Groups](http://groups.google.com/group/packer-tool)

### Getting Support

If you believe your build is failing as a result of a bug in Terraform
Enterprise, or would like other support, please
[email us](mailto:support@hashicorp.com).
