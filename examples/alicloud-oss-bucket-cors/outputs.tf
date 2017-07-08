output "bucket-cors" {
  value = "${alicloud_oss_bucket.bucket-cors.id}"
}

output "bucket-cors-rule" {
  value = "${alicloud_oss_bucket.bucket-cors.cors_rule}"
}