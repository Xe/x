terraform {
  backend "s3" {
    bucket = "within-tf-state"
    key    = "k8s/aeacus/external-dns"
    region = "us-east-1"
  }
}

resource "aws_dynamodb_table" "external_dns_crd" {
  name           = "external-dns-crd-aeacus"
  billing_mode   = "PROVISIONED"
  read_capacity  = 1
  write_capacity = 1
  table_class    = "STANDARD"

  attribute {
    name = "k"
    type = "S"
  }

  hash_key = "k"
}

resource "aws_dynamodb_table" "external_dns_ingress" {
  name           = "external-dns-ingress-aeacus"
  billing_mode   = "PROVISIONED"
  read_capacity  = 1
  write_capacity = 1
  table_class    = "STANDARD"

  attribute {
    name = "k"
    type = "S"
  }

  hash_key = "k"
}
