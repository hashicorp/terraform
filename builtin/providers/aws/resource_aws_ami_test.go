package aws

// FIXME: The aws_ami resource doesn't currently have any acceptance tests,
// since creating an AMI requires having an EBS snapshot and we don't yet
// have a resource type for creating those.
// Once there is an aws_ebs_snapshot resource we can use it to implement
// a reasonable acceptance test for aws_ami. Until then it's necessary to
// test manually using a pre-existing EBS snapshot.
