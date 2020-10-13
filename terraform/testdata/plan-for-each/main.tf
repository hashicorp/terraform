# maps
resource "aws_instance" "foo" {
    for_each = {
        a = "thing"
        b = "another thing"
        c = "yet another thing"
    }
    num = "3"
}

# sets
resource "aws_instance" "bar" {
    for_each = toset([])
}
resource "aws_instance" "bar2" {
    for_each = toset(list("z", "y", "x"))
}

# an empty map should generate no resource
resource "aws_instance" "baz" {
    for_each = {}
}

# references
resource "aws_instance" "boo" {
    foo = aws_instance.foo["a"].num
}

resource "aws_instance" "bat" {
    for_each = {
        my_key = aws_instance.boo.foo
    }
    foo = each.value
}

