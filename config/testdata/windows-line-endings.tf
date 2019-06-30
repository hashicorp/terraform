// This is a comment
resource "aws_instance" "test" {
    user_data = <<HEREDOC
    test script
HEREDOC
}
