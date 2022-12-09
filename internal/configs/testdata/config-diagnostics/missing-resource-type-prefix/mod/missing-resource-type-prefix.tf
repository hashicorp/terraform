# Sometimes people new to Terraform copy the provider examples incorrectly
# and omit or mistype the prefix that implies the provider that the resource
# should belong to.
#
# This test is covering this situation to make sure we continue to return
# a reasonably helpful error message for it, which in particular mentions that
# Terraform is trying guessing the intended provider name from the prefix.

resource "s3_bucket" "example" {
}
