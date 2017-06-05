output "bucket-new" {
  value = "${alicloud_oss_bucket.bucket-new.id}"
}

output "bucket-attr" {
  value = "${alicloud_oss_bucket.bucket-attr.id}"
}

output "bucket-attr-website" {
  value = "${alicloud_oss_bucket.bucket-attr.website}"
}

output "bucket-attr-logging" {
  value = "${alicloud_oss_bucket.bucket-attr.logging}"
}

output "bucket-attr-lifecycle" {
  value = "${alicloud_oss_bucket.bucket-attr.lifecycle}"
}

output "bucket-attr-referers" {
  value = "${alicloud_oss_bucket.bucket-attr.referer_config}"
}