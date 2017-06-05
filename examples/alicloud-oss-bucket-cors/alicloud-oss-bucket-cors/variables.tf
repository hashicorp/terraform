variable "name" {
  default = "bucket-us-20170509"
}

variable "acl" {
  default = "public-read"
}

variable "allow-origins-star" {
  default = "*"
}

variable "allow-origins-aliyun" {
  default = "http://www.aliyun.com, http://*.aliyun.com"
}

variable "allow-methods-get" {
  default = "GET"
}

variable "allow-methods-put" {
  default = "PUT,GET"
}

variable "allowed_headers" {
  default = "authorization"
}

variable "expose_headers" {
  default = "x-oss-test, x-oss-test1"
}

variable "max_age_seconds" {
  default = 100
}