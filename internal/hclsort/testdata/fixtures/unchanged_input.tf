resource "aws_instance" "unchanged" {
  for_each = [1]

  ami           = "ami-12345678"
  instance_type = "t4g.nano"
  tags = {
    Name = "test-spot"
  }

  instance_market_options {
    market_type = "spot"
  }

  lifecycle {
    ignore_changes = [tags]
  }

  depends_on = [aws_instance.other]
}
