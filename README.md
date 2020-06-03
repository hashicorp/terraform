# Terraform v0.13 Beta Guide

Hello! This temporary repository contains some hopefully-helpful resources for
people participating in the Terraform v0.13 beta testing process. It highlights
the major changes coming in Terraform v0.13 and includes some configuration
examples you might use as a starting point for testing. Please play with the
examples and try to adapt them to patterns you are using in your real
infrastructure!

> **We do not recommend using beta releases in production**. While we have
> performed some internal alpha testing prior to this release, the betas will
> be the first exposure of some of this code to use-cases the Terraform team
> didn't anticipate during that internal testing, and so there may well be bugs
> lurking which we'll aim to address during the beta period.

The information in this guide is likely to be removed or become stale after
the v0.13 beta period concludes, so we don't recommend using this content
as an ongoing reference resource. If you are reading this after v0.13 final is
released, please refer to
[the main Terraform documentation](https://www.terraform.io/docs/cli-index.html)
for up-to-date information.

---

## Terraform v0.13 Highlights

While we do welcome feedback of any sort during the beta process, our
development efforts during this period will be focused mainly on fixing issues
related to the main feature development themes for this release and so issues
that pre-existed in Terraform v0.12 or earlier are likely to be deferred to a
later time.

The main new features and significant changes in this release are:

* [`for_each` and `count` for modules](./module-repetition)
* [`depends_on` for modules](./module-depends)
* [Automatic installation of third-party providers](./provider-sources)
* [Custom validation rules for module variables](./variable-validation)

The above directories contain Terraform configurations demonstrating the
features. You can clone the contents of this guide into a local directory to
try them out with the following command:

```
git clone https://github.com/hashicorp/terraform.git -b guide-v0.13-beta --single-branch terraform-v0.13-beta
```

## Draft Upgrade Guide

When we release the final version of v0.13.0 we will, as usual, include an
upgrade guide on the website to cover some different situations that users
might encounter when upgrading from Terraform v0.12.

We have
[a draft version of the upgrade guide](./draft-upgrade-guide.md)
here for your reference. Please keep in mind that the content in there is
written with a different audience in mind than the rest of the content
here, in that this will be the guide for those who are intending to upgrade
to v0.13 and _keep using it_, but we don't recommend doing that during the
beta period.

## How to Send Feedback

If you try the beta and run into bugs or confusing behavior, we'd love to hear
from you!

Please check
[the known issues for v0.13](https://github.com/hashicorp/terraform/issues?q=is%3Aissue+label%3Av0.13)
first to see if someone has already reported a similar problem. In order to
expedite the bug-fixing progress it's helpful to have different examples of the
same problem collected together in the same issue.

If you can't find an existing issue covering the situation you've encountered,
then please [open a new issue](https://github.com/hashicorp/terraform/issues/new/choose)
describing the problem, and be sure to mention in the issue body that you
are using a v0.13 beta version and indicate exactly which release you are
using: v0.13-beta1 as opposed to v0.13-beta2, for example. Our team will then
label the issue to include it in the known issues list mentioned above.

We'll be focusing mainly on high-priority bugs during the beta period in order
to ensure a high-quality final release, so we'd ask that new feature requests
or enhancement requests related to the features wait until after v0.13.0 final
is released and then we can potentially address them in later releases.

Thank you for your interest in the v0.13 beta process!
