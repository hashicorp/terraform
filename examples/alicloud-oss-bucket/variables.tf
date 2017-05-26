variable "bucket-new" {
  default = "bucket-20170509-1"
}

variable "bucket-attr"{
  default = "bucket-20170509-2"
}

variable "acl-bj" {
  default = "public-read"
}

variable "index-doc" {
  default = "index.html"
}

variable "error-doc" {
  default = "error.html"
}

variable "target-prefix" {
  default = "log/"
}

variable "role-days" {
  default = "expirationByDays"
}

variable "rule-days" {
  default = 365
}

variable "role-date" {
  default = "expirationByDate"
}

variable "rule-date" {
  default = "2018-01-01"
}

variable "rule-prefix" {
  default = "path"
}

variable "allow-empty" {
  default = true
}

variable "referers" {
  default = "http://www.aliyun.com, https://www.aliyun.com, http://?.aliyun.com"
}