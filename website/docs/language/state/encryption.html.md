---
layout: "language"
page_title: "State: Client-Side Remote State Encryption"
sidebar_current: "docs-state-encryption"
description: |-
  Client-Side Remote State Encryption.
---

# Client-Side Remote State Encryption

State regularly contains mission-critical sensitive information such as credentials.

Several remote state backends already support server-side encryption, but

- many do not
- some implementations effectively turn encryption into nothing but a proxy for key access control  
- you need to trust the remote backend operator
- you need to trust your communication channel to the remote backend

This **experimental feature** lets you encrypt the complete state on the client before transferring it.

Depending on your choice of state crypto provider, no third party ever sees the key.

## Limitations

Client-side state encryption will not work with [enhanced backends](/docs/language/settings/backends/index.html), as 
those need access to the information in the state to function correctly.

**Client-side remote state encryption is an experimental feature!** Implementation and configuration details are subject to change, 
and not all remote state backends have been fully tested!

## State Crypto Providers

State crypto providers are a transparent encryption layer around the communication with remote state backends.

**This is an experimental feature!**

You can configure state crypto providers by using two environment variables:

- `TF_REMOTE_STATE_ENCRYPTION`: configuration used for encryption and decryption
- `TF_REMOTE_STATE_DECRYPTION_FALLBACK`: fallback configuration for decryption, tried if decryption fails with the first choice  

You set each of these to a json document with two fields

  * `implementation`: select a _state crypto provider_ by name
  * `parameters`: configure the _state crypto provider_

to enable client-side remote state encryption. To disable, either do not set the variable at all or set it to a blank value.

Right now, there is only one value for `implementation`: `client-side/AES256-CFB/SHA256`. 

In the future, more state encryption providers may be added, such as:

- asymmetric encryption with RSA public key cryptography
- key retrieval from [Vault](https://www.vaultproject.io/)
- ...

### client-side/AES256-CFB/SHA256

This state crypto provider offers pure client-side symmetric encryption. 

The key is not transferred to any third party. Note that this places the burden of key management on you
and you alone.

Encryption is performed with AES256-CFB, using a fresh random initialization vector every time. Payload integrity
is verified using a SHA256 hash over the plaintext, which is encrypted with the plaintext.

_Implementation Name:_ `client-side/AES256-CFB/SHA256`

_Parameters:_

- `key`: the 32 byte AES256 key represented in hexadecimal, must be exactly 64 characters, `0-9a-f` only.

Example:

```shell
export TF_REMOTE_STATE_ENCRYPTION='{"implementation":"client-side/AES256-CFB/SHA256","parameters":{"key":"a0a1a2a3a4a5a6a7a8a9b0b1b2b3b4b5b6b7b8b9c0c1c2c3c4c5c6c7c8c9d0d1"}}'
```

## State Encryption Lifecycle

When transparently encrypting the state, one must consider these lifecycle events:

- initial encryption
- the way out (decrypting state that is currently encrypted)
- key rotation
- switching state encryption providers (e.g. there is a security issue with one of them, or a wish to migrate)

### Preparations

All remote state encryption lifecycle events will require you to cause a change in state to trigger (re-)encryption 
or decryption during `terraform apply`.

The easiest way to achieve this is by putting a null resource in your state that you do not use anywhere else.
Then you can change its value to an arbitrary new value without needing to make an actual change to your infrastructure.

### Initial Encryption

Let us assume you currently have unencrypted remote state and wish to encrypt it going forward. 

The same approach also works when remote state is initially created. 

With the ability to force a change in state in place (see Preparations above) do this:

1. Set `TF_REMOTE_STATE_ENCRYPTION` to the desired configuration
2. Leave `TF_REMOTE_STATE_DECRYPTION_FALLBACK` unset
3. Cause a change in state and run `terraform apply`. This will encrypt your state.

From now on, you will need to run terraform with `TF_REMOTE_STATE_ENCRYPTION` set to the configuration you just used, 
or it will not be able to read your state any more.

### Permanent Decryption

Now let us assume that you wish to move from encrypted remote state to the default unencrypted state. Do this:

1. Leave `TF_REMOTE_STATE_ENCRYPTION` unset
2. Set `TF_REMOTE_STATE_DECRYPTION_FALLBACK` to the configuration previously used to encrypt your state
3. Cause a change in state and run `terraform apply`. This will decrypt your state.

Once all your state has been decrypted, you should unset both environment variables, and as long as you
do not set them again, terraform will operate on unencrypted remote state.

### Key Rotation

Now let us assume that you wish to move from one encryption key to another. Do this:

1. Set `TF_REMOTE_STATE_ENCRYPTION` to the configuration with the new key
2. Set `TF_REMOTE_STATE_DECRYPTION_FALLBACK` to the previous configuration with the old key
3. Cause a change in state and run `terraform apply`. This will re-encrypt your state.

Once all your state has been migrated, you can then drop the fallback configuration and run
terraform with `TF_REMOTE_STATE_ENCRYPTION` set to the new configuration going forward.

### Switching State Crypto Providers 

Just use the same approach as for key rotation. There is no requirement to use the same state crypto
provider for encryption and fallback.
