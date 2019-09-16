resource "aws_instance" "web" {
    // require_new is a special attribute recognized by testDiffFn that forces
    // a new resource on every apply
    require_new = "yes"
    lifecycle {
        create_before_destroy = true
    }
}
