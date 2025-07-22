# Package cloudplugin Test Data Signing Key

This directory contains a private key that is only used for signing the test data, along with the public key that the package uses to verfify the signing. Here are the steps to reproduce the test data, which would be necessary if the archive and checksum changes.

1. Import the secret key

`gpg --import sample.private.key`

2. Sign the sample_release SHA256SUMS file using the sample key:

`gpg -u 200BDA882C95B80A --output archives/terraform-cloudplugin_0.1.0_SHA256SUMS.sig --detach-sig archives/terraform-cloudplugin_0.1.0_SHA256SUMS`
