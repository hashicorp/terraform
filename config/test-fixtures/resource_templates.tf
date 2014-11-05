resource_template "aws-web" {
    ami = "ami-408c7f28"
    instance_type = "t1.micro"
    key_name = "web-key"
    availability_zone = "us-west-2"
    subnet_id = "subnet-9d4a7b6c"
}

resource "aws_instance" "web1" {
    resource_template = "aws-web"
}

resource "aws_instance" "web2" {
    resource_template = "aws-web"
}