output "bucket-new" {
  value = "${alicloud_oss_bucket.bucket-new.id}"
}

output "content" {
  value = "${alicloud_oss_bucket_object.content.id}"
}

