resource "aws_route53_zone" "yada" {

}

resource "aws_route53_zone" "terra" {
	count = 2
}
