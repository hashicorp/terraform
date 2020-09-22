resource "null_resource" "a" {
  provisioner "local-exec" {
    when       = "create" # WARNING: Quoted keywords are deprecated
    on_failure = "fail"   # WARNING: Quoted keywords are deprecated
  }
}
