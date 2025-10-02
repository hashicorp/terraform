# These run blocks either:
#  1) don't set an explicit state_key value and test the working directory,
#     so would have the same internal state file as run blocks in the other test file.
#  2) do set an explicit state_key, which matches run blocks in the other test file.
# 
# test_file_two.tftest.hcl as the same content as test_file_one.tftest.hcl,
# with renamed run blocks.
run "file_2_load_state" {
    backend "local" {
        path = "state/terraform.tfstate"
    }
}

run "file_2_test" {
    assert {
        condition = aws_instance.web.ami == "ami-1234"
        error_message = "AMI should be ami-1234"
    }
}

run "file_2_load_state_state_key" {
    state_key = "foobar"
    backend "local" {
        path = "state/terraform.tfstate"
    }
}

run "file_2_test_state_key" {
    state_key = "foobar"
    assert {
        condition = aws_instance.web.ami == "ami-1234"
        error_message = "AMI should be ami-1234"
    }
}
