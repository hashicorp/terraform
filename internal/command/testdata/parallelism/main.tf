resource "test0_instance" "foo" {}
resource "test1_instance" "foo" {}
resource "test2_instance" "foo" {}
resource "test3_instance" "foo" {}
resource "test4_instance" "foo" {}
resource "test5_instance" "foo" {}
resource "test6_instance" "foo" {}
resource "test7_instance" "foo" {}
resource "test8_instance" "foo" {}
resource "test9_instance" "foo" {}

terraform {
  required_providers {
    test0 = { source = "hashicorp/test0" }
    test1 = { source = "hashicorp/test1" }
    test2 = { source = "hashicorp/test2" }
    test3 = { source = "hashicorp/test3" }
    test4 = { source = "hashicorp/test4" }
    test5 = { source = "hashicorp/test5" }
    test6 = { source = "hashicorp/test6" }
    test7 = { source = "hashicorp/test7" }
    test8 = { source = "hashicorp/test8" }
    test9 = { source = "hashicorp/test9" }
  }
}
