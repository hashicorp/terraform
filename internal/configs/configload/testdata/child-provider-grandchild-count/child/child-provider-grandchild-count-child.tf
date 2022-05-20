provider "boop" {
    blah = true
}

module "grandchild" {
    source = "../grandchild"

    # grandchild's caller (this file) has a legacy nested provider block, but
    # grandchild itself does not and so it's valid to use "count" here even
    # though it wouldn't be valid to call "child" (this file) with "count".
    count = 2
}
