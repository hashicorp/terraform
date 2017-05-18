---
layout: "enterprise"
page_title: "Authentication - Accounts - Terraform Enterprise"
sidebar_current: "docs-enterprise-accounts-authentication"
description: |-
  Terraform Enterprise requires a username and password to sign up and login. However, there are several ways to authenticate with your account.
---

# Authentication

Terraform Enterprise requires a username and password to sign up and login.
However, there are several ways to authenticate with your account.

### Authentication Tokens

Authentication tokens are keys used to access your account via tools or over the
various APIs used in Terraform Enterprise.

You can create new tokens in the token section of your account settings. It's
important to keep tokens secure, as they are essentially a password and can be
used to access your account or resources. Additionally, token authentication
bypasses two factor authentication.

### Authenticating Tools

All HashiCorp tools look for the `ATLAS_TOKEN` environment variable:

```shell
$ export ATLAS_TOKEN=TOKEN
```

This will automatically authenticate all requests against this token. This is
the recommended way to authenticate with our various tools. Care should be given
to how this token is stored, as it is as good as a password.

### Two Factor Authentication

You can optionally enable Two Factor authentication, requiring an SMS or TOTP
one-time code every time you log in, after entering your username and password.

You can enable Two Factor authentication in the security section of your account
settings.

Be sure to save the generated recovery codes. Each backup code can be used once
to sign in if you do not have access to your two-factor authentication device.

### Sudo Mode

When accessing certain admin-level pages (adjusting your user profile, for
example), you may notice that you're prompted for your password, even though
you're already logged in. This is by design, and aims to help guard protect you
if your screen is unlocked and unattended.

### Session Management

You can see a list of your active sessions on your security settings page. From
here, you can revoke sessions, in case you have lost access to a machine from
which you were accessing.
