terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 3.0"
    }
  }
}

provider "aws" {
  profile = "vendor_root"
  region  = "eu-central-1"

  default_tags {
    tags = {
      Owner   = "Shay"
      Project = "ExerciseSender"
    }
  }
}

locals {
  src_path     = "./cmd/lambda/main.go"
  binary_name  = "./lambda"
  binary_path  = "./bin/${local.binary_name}"
  archive_path = "./bin/${local.binary_name}.zip"
}

resource "aws_vpc" "my-vpc" {
  cidr_block           = "10.0.0.0/16"
  instance_tenancy     = "default"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "my-vpc"
  }
}

resource "aws_ecr_repository" "my-repo" {
  name = "my-repo"
}

resource "aws_ecs_cluster" "my-cluster" {
  name = "my-cluster"
}

resource "aws_cloudwatch_log_group" "task-log-group" {
  name = "awslogs-my-task"
}

resource "aws_ecs_task_definition" "my-task" {
  family = "my-task"
  // See: https://docs.aws.amazon.com/AmazonECS/latest/userguide/task_definition_parameters.html
  container_definitions    = <<DEFINITION
  [
    {
      "name": "my-task",
      "image": "${aws_ecr_repository.my-repo.repository_url}:latest",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080,
          "hostPort": 8080
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-create-group": "true",
          "awslogs-group": "awslogs-my-task",
          "awslogs-region": "eu-central-1",
          "awslogs-stream-prefix": "awslogs-my-task"
        }
      },
      "memory": 512,
      "cpu": 256
    }
  ]
  DEFINITION
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = "256"
  memory                   = "512"
  // This is the role of the "host" - what starts the container
  execution_role_arn = aws_iam_role.ecs-task-execution-role.arn
  // This is the role of the "task" - what the container does
  task_role_arn = aws_iam_role.task-role.arn
}



resource "aws_iam_role" "ecs-task-execution-role" {
  name               = "ecs-task-execution-role"
  assume_role_policy = data.aws_iam_policy_document.assume_role_policy.json
}

data "aws_iam_policy_document" "assume_role_policy" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["ecs-tasks.amazonaws.com"]
    }
  }
}

resource "aws_iam_role_policy_attachment" "ecs-task-execution-role-policy" {
  role       = aws_iam_role.ecs-task-execution-role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_internet_gateway" "my-gateway" {
  // This connects the gateway to the VPC
  vpc_id = aws_vpc.my-vpc.id

  tags = {
    PublicFacing = "true"
  }
}

// Define the subnets: we need 2 public subnets and 2 private subnets
// The public subnets will be used for the load balancer and the private subnets will be used for the ECS tasks
resource "aws_subnet" "public1" {
  vpc_id            = aws_vpc.my-vpc.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = "eu-central-1a"
  // public subnet - public IP
  map_public_ip_on_launch = true
  tags = {
    Name         = "public1"
    PublicFacing = "true"
  }
}

resource "aws_subnet" "public2" {
  vpc_id            = aws_vpc.my-vpc.id
  cidr_block        = "10.0.2.0/24"
  availability_zone = "eu-central-1b"
  // public subnet - public IP
  map_public_ip_on_launch = true
  tags = {
    Name         = "public2"
    PublicFacing = "true"
  }
}

// Add the internet gateway to the public subnets
resource "aws_route_table" "public1" {
  vpc_id = aws_vpc.my-vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.my-gateway.id
  }

  tags = {
    Name = "public1"
  }
}

resource "aws_route_table_association" "public1" {
  subnet_id      = aws_subnet.public1.id
  route_table_id = aws_route_table.public1.id
}

resource "aws_route_table" "public2" {
  vpc_id = aws_vpc.my-vpc.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.my-gateway.id
  }

  tags = {
    Name = "public2"
  }
}

resource "aws_route_table_association" "public2" {
  subnet_id      = aws_subnet.public2.id
  route_table_id = aws_route_table.public2.id
}

resource "aws_subnet" "private1" {
  vpc_id            = aws_vpc.my-vpc.id
  cidr_block        = "10.0.11.0/24"
  availability_zone = "eu-central-1a"
  // private subnet - no public IP
  map_public_ip_on_launch = false
  tags = {
    Name = "private1"
  }
}
resource "aws_subnet" "private2" {
  vpc_id            = aws_vpc.my-vpc.id
  cidr_block        = "10.0.12.0/24"
  availability_zone = "eu-central-1b"
  // private subnet - no public IP
  map_public_ip_on_launch = false
  tags = {
    Name = "private2"
  }
}

resource "aws_alb" "my-alb" {
  name               = "my-alb"
  load_balancer_type = "application"

  subnets = [
    "${aws_subnet.public1.id}",
    "${aws_subnet.public2.id}"
  ]

  # Referencing the security group
  security_groups = ["${aws_security_group.alb-security-group.id}"]
}

# Creating a security group for the load balancer:
resource "aws_security_group" "alb-security-group" {
  name        = "alb-security-group"
  description = "Allow traffic to the ALB from anywhere on 8080."
  vpc_id      = aws_vpc.my-vpc.id

  ingress {
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"] # Allowing traffic in from all sources
  }

  egress {
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = ["${aws_security_group.ecs-web-service-security-group.id}"]
  }
}

resource "aws_lb_target_group" "my-target-group-new" {
  name        = "target-group-new"
  port        = 8080
  protocol    = "HTTP"
  target_type = "ip"
  vpc_id      = aws_vpc.my-vpc.id

  health_check {
    matcher = "200"
    path    = "/health"
  }
}

resource "aws_lb_listener" "my-listener" {
  load_balancer_arn = aws_alb.my-alb.arn
  port              = "8080"
  protocol          = "HTTP"
  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.my-target-group-new.arn
  }
}

resource "aws_security_group" "ecs-web-service-security-group" {
  name        = "ecs-web-service-security-group"
  description = "Allow traffic to the ECS web service"
  vpc_id      = aws_vpc.my-vpc.id

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_ecs_service" "web-service" {
  name            = "web-service"
  cluster         = aws_ecs_cluster.my-cluster.id
  task_definition = aws_ecs_task_definition.my-task.arn
  desired_count   = 3
  launch_type     = "FARGATE"

  network_configuration {
    subnets         = ["${aws_subnet.private1.id}", "${aws_subnet.private2.id}"]
    security_groups = ["${aws_security_group.ecs-web-service-security-group.id}"]
    // Will not assign public IP to the tasks - they will be in a private subnet,
    // and the load balancer will be in a public subnet
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.my-target-group-new.arn
    container_name   = aws_ecs_task_definition.my-task.family
    container_port   = 8080
  }
}

// Add VPC endpoints for ECR in the private subnets
resource "aws_security_group" "private-to-ecr-vpc-security-group" {
  name        = "private-to-ecr-vpc-security-group"
  description = "Allow traffic to the ECR VPC endpoint"
  vpc_id      = aws_vpc.my-vpc.id

  ingress {
    from_port       = 443
    to_port         = 443
    protocol        = "tcp"
    security_groups = ["${aws_security_group.ecs-web-service-security-group.id}"]
  }
}
resource "aws_vpc_endpoint" "private-to-ecr-dkr" {
  vpc_id              = aws_vpc.my-vpc.id
  service_name        = "com.amazonaws.eu-central-1.ecr.dkr"
  vpc_endpoint_type   = "Interface"
  private_dns_enabled = "true"
  security_group_ids  = ["${aws_security_group.private-to-ecr-vpc-security-group.id}"]
  subnet_ids          = ["${aws_subnet.private1.id}", "${aws_subnet.private2.id}"]
}

resource "aws_vpc_endpoint" "private-to-ecr-api" {
  vpc_id              = aws_vpc.my-vpc.id
  service_name        = "com.amazonaws.eu-central-1.ecr.api"
  vpc_endpoint_type   = "Interface"
  private_dns_enabled = "true"
  security_group_ids  = ["${aws_security_group.private-to-ecr-vpc-security-group.id}"]
  subnet_ids          = ["${aws_subnet.private1.id}", "${aws_subnet.private2.id}"]
}

resource "aws_vpc_endpoint" "private-to-logs" {
  vpc_id              = aws_vpc.my-vpc.id
  service_name        = "com.amazonaws.eu-central-1.logs"
  vpc_endpoint_type   = "Interface"
  private_dns_enabled = "true"
  security_group_ids  = ["${aws_security_group.private-to-ecr-vpc-security-group.id}"]
  subnet_ids          = ["${aws_subnet.private1.id}", "${aws_subnet.private2.id}"]
}

resource "aws_vpc_endpoint" "private-to-secretmanager" {
  vpc_id              = aws_vpc.my-vpc.id
  service_name        = "com.amazonaws.eu-central-1.secretsmanager"
  vpc_endpoint_type   = "Interface"
  private_dns_enabled = "true"
  security_group_ids  = ["${aws_security_group.private-to-ecr-vpc-security-group.id}"]
  subnet_ids          = ["${aws_subnet.private1.id}", "${aws_subnet.private2.id}"]
}

// Create the Amazon S3 Gateway endpoint. This is required since the image
// layers are stored in Amazon S3.

resource "aws_route_table" "my-route-table" {
  vpc_id = aws_vpc.my-vpc.id
}


resource "aws_route_table_association" "private1" {
  subnet_id      = aws_subnet.private1.id
  route_table_id = aws_route_table.my-route-table.id
}

resource "aws_route_table_association" "private2" {
  subnet_id      = aws_subnet.private2.id
  route_table_id = aws_route_table.my-route-table.id
}

# associate route table with VPC endpoint
resource "aws_vpc_endpoint_route_table_association" "route_table_association_s3" {
  route_table_id  = aws_route_table.my-route-table.id
  vpc_endpoint_id = aws_vpc_endpoint.private-to-s3.id
}

resource "aws_vpc_endpoint" "private-to-s3" {
  vpc_id            = aws_vpc.my-vpc.id
  vpc_endpoint_type = "Gateway"
  service_name      = "com.amazonaws.eu-central-1.s3"
  route_table_ids = [
    "${aws_route_table.my-route-table.id}"
  ]
}

resource "aws_vpc_endpoint" "private-to-dynamodb" {
  vpc_id            = aws_vpc.my-vpc.id
  service_name      = "com.amazonaws.eu-central-1.dynamodb"
  vpc_endpoint_type = "Gateway"
  route_table_ids = [
    "${aws_route_table.my-route-table.id}"
  ]
}

resource "aws_vpc_endpoint_route_table_association" "route_table_association_dynamodb" {
  route_table_id  = aws_route_table.my-route-table.id
  vpc_endpoint_id = aws_vpc_endpoint.private-to-dynamodb.id
}

resource "aws_dynamodb_table" "my-db" {
  name         = "my-db"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "requestId"

  // Need streaming for the lambda
  stream_enabled   = true
  stream_view_type = "NEW_IMAGE"

  attribute {
    name = "requestId"
    type = "S"
  }
}

resource "aws_iam_role" "task-role" {
  name               = "task-role"
  assume_role_policy = data.aws_iam_policy_document.assume_role_policy.json

  inline_policy {
    name = "task-policy"
    policy = jsonencode({
      "Version" : "2012-10-17",
      "Statement" : [
        {
          "Effect" : "Allow",
          "Action" : "dynamodb:PutItem",
          "Resource" : "${aws_dynamodb_table.my-db.arn}"
        }
      ]
    })
  }
}

// Create a role for the lambda
// allow lambda service to assume (use) the role with such policy
data "aws_iam_policy_document" "my-assume_lambda_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

// create lambda role, that lambda function can assume (use)
resource "aws_iam_role" "my-lambda" {
  name               = "AssumeLambdaRole"
  description        = "Role for lambda to assume lambda"
  assume_role_policy = data.aws_iam_policy_document.my-assume_lambda_role.json
}

data "aws_iam_policy_document" "allow_lambda_outputs" {
  statement {
    effect = "Allow"
    actions = [
      "s3:PutObject"
    ]

    resources = [
      "${aws_s3_bucket.my-lambda-bucket.arn}",
      "${aws_s3_bucket.my-lambda-bucket.arn}/*",
    ]
  }

  statement {
    effect = "Allow"
    actions = [
      "sns:Publish"
    ]
    resources = ["*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents"
    ]

    resources = [
      "arn:aws:logs:eu-central-1:*:*"
    ]
  }

  statement {
    effect = "Allow"
    actions = [
      "dynamodb:DescribeStream",
      "dynamodb:GetRecords",
      "dynamodb:GetShardIterator",
      "dynamodb:ListStreams"
    ]

    resources = [
      "${aws_dynamodb_table.my-db.arn}",
      // without this line, getting IAM error on getting the streams in the event sourcing
      "${aws_dynamodb_table.my-db.arn}/*"
    ]
  }
}

// create a policy to allow writing into logs and create logs stream
resource "aws_iam_policy" "function_outputs_policy" {
  name        = "AllowLambdaOutputs"
  description = "Policy "
  policy      = data.aws_iam_policy_document.allow_lambda_outputs.json
}

// attach policy to our created lambda role
resource "aws_iam_role_policy_attachment" "lambda_outputs_policy_attachment" {
  role       = aws_iam_role.my-lambda.id
  policy_arn = aws_iam_policy.function_outputs_policy.arn
}

// Hack to always rebuild the lambda function
resource "null_resource" "always_run" {
  triggers = {
    timestamp = "${timestamp()}"
  }
}

// build the binary for the lambda function in a specified path
resource "null_resource" "function_binary" {
  provisioner "local-exec" {
    command = "GOOS=linux GOARCH=amd64 CGO_ENABLED=0 GOFLAGS=-trimpath go build -mod=readonly -ldflags='-s -w' -o ${local.binary_path} ${local.src_path}"
  }

  lifecycle {
    replace_triggered_by = [null_resource.always_run]
  }
}

// zip the binary, as we can use only zip files to AWS lambda
data "archive_file" "function_archive" {
  depends_on = [null_resource.function_binary]

  type        = "zip"
  source_file = local.binary_path
  output_path = local.archive_path
}

// create the lambda function from zip file
resource "aws_lambda_function" "my-function" {
  function_name = "listen-to-dynamodb-and-send-sms"
  description   = "This function listens to a DynamoDB stream. Whenever it's triggered, it saves something to S3 and sends an SMS message."
  role          = aws_iam_role.my-lambda.arn
  handler       = local.binary_name
  memory_size   = 128

  filename         = local.archive_path
  source_code_hash = data.archive_file.function_archive.output_base64sha256

  runtime = "go1.x"

  environment {
    variables = {
      AWS_EXERCISE_BUCKET_NAME = aws_s3_bucket.my-lambda-bucket.id
      AWS_EXERCISE_LOG_FORMAT  = "json"
    }
  }
}

// Create s3 bucket for lambda output
resource "aws_s3_bucket" "my-lambda-bucket" {
  bucket = "shay-nehmad-private-bucket"

  tags = {
    Name = "shay-nehmad-private-bucket"
  }
}

resource "aws_lambda_event_source_mapping" "dynamodb-stream-to-lambda-trigger" {
  event_source_arn  = aws_dynamodb_table.my-db.stream_arn
  function_name     = aws_lambda_function.my-function.function_name
  starting_position = "LATEST"
  batch_size        = 1
}
