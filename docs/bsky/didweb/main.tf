terraform {
  backend "s3" {
    bucket = "within-tf-state"
    key    = "shitposting/bsky-pds"
    region = "us-east-1"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }

    civo = {
      source  = "civo/civo"
      version = "1.1.3"
    }

    tigris = {
      source  = "tigrisdata/tigris"
      version = "1.0.4"
    }
  }
}

provider "civo" {
  region = "nyc1"
}

provider "tigris" {
  access_key = var.tigris_access_key_id
  secret_key = var.tigris_secret_access_key
}