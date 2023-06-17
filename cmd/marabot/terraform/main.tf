terraform {
  backend "s3" {
    bucket = "within-tf-state"
    key    = "aws/marabot"
    region = "us-east-1"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 4.0"
    }
  }
}

resource "aws_s3_bucket" "marabot" {
  bucket = "xeserv-marabot"

  tags = {
    Name   = "Marabot uploads"
  }
}

resource "aws_s3_bucket_lifecycle_configuration" "marabot" {
  bucket = aws_s3_bucket.marabot.id

  rule {
    id      = "auto_glacier_backup"
    status  = "Enabled"

    transition {
      days          = 90
      storage_class = "GLACIER"
    }

    filter {
      prefix = "attachments/"
    }
  }
}

data "aws_iam_policy_document" "marabot" {
  statement {
    actions = [
      "s3:GetObject",
      "s3:PutObject",
      "s3:ListBucket",
    ]
    effect = "Allow"
    resources = [
      aws_s3_bucket.marabot.arn,
      "${aws_s3_bucket.marabot.arn}/*",
    ]
  }

  statement {
    actions   = ["s3:ListAllMyBuckets"]
    effect    = "Allow"
    resources = ["*"]
  }
}

resource "aws_iam_policy" "marabot" {
  name        = "marabot-policy"
  description = "policy for managing S3 for marabot"

  policy = data.aws_iam_policy_document.marabot.json
}

resource "aws_iam_user" "marabot" {
  name = "marabot"
  path = "/within/marabot/"
}

resource "aws_iam_user_policy_attachment" "marabot" {
  user       = aws_iam_user.marabot.name
  policy_arn = aws_iam_policy.marabot.arn
}

resource "aws_iam_access_key" "creds" {
  user = aws_iam_user.marabot.name
}

output "marabot_environment_file" {
  value = <<EOT
AWS_ACCESS_KEY_ID=${aws_iam_access_key.creds.id}
AWS_SECRET_ACCESS_KEY=${nonsensitive(aws_iam_access_key.creds.secret)}
EOT
}
