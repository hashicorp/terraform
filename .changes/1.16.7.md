## 1.16.7 (October 22, 2026)


BUG FIXES:

* command/test: Avoid panic when testing schema w/ deprecated values ([#39406](https://github.com/hashicorp/terraform/issues/39406))


NOTES:

* Terraform is now built with Go 1.26.10, which no longer trusts certificates issued through a set of compromised eMudhra intermediate CAs (CVE-2026-111943) ([#39417](https://github.com/hashicorp/terraform/issues/39417))


