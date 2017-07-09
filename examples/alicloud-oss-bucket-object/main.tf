provider "alicloud" {
  alias = "bj-prod"
  region = "cn-beijing"
}

resource "alicloud_oss_bucket" "bucket-new" {
  bucket = "${var.bucket-new}"
  acl = "${var.acl}"
}


resource "alicloud_oss_bucket_object" "content" {
  bucket = "${alicloud_oss_bucket.bucket-new.bucket}"
  key = "${var.object-key}"
  content = "${var.object-content}"
}
