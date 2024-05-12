terraform {
  backend "s3" {
    bucket = "within-tf-state"
    key    = "ingressd"
    region = "us-east-1"
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }

    cloudinit = {
      source  = "hashicorp/cloudinit"
      version = "2.3.4"
    }

    tailscale = {
      source  = "tailscale/tailscale"
      version = "0.16.1"
    }

    vultr = {
      source  = "vultr/vultr"
      version = "2.19.0"
    }
  }
}

provider "tailscale" {
  tailnet = "cetacean.org.github"
}

provider "vultr" {
  rate_limit  = 100
  retry_limit = 3
}

data "vultr_os" "rocky" {
  filter {
    name   = "name"
    values = ["Rocky Linux 9 x64"]
  }
}

data "vultr_plan" "ingressd" {
  filter {
    name   = "id"
    values = ["vc2-1c-1gb"]
  }
}

resource "tailscale_tailnet_key" "ingressd" {
  reusable      = true
  ephemeral     = false
  preauthorized = true
  description   = "ingressd key"
  tags          = ["tag:alrest"]
}

data "aws_route53_zone" "cetacean_club" {
  name = "cetacean.club."
}

resource "vultr_instance" "my_instance" {
  plan                = "vc2-1c-2gb"
  region              = "yto"
  os_id               = data.vultr_os.rocky.id
  label               = "ingressd"
  tags                = ["rocky"]
  hostname            = "ingressd"
  enable_ipv6         = true
  disable_public_ipv4 = false
  backups             = "enabled"
  backups_schedule {
    type = "daily"
  }
  ddos_protection  = false
  activation_email = true
  user_data        = <<EOF
#!/bin/sh

curl -fsSL https://tailscale.com/install.sh | sh
tailscale up --authkey ${resource.tailscale_tailnet_key.ingressd.key} --ssh --accept-routes
    EOF
}

resource "aws_route53_record" "A" {
  zone_id = data.aws_route53_zone.cetacean_club.zone_id
  name    = "ingressd.${data.aws_route53_zone.cetacean_club.name}"
  type    = "A"
  ttl     = "300"
  records = [resource.vultr_instance.my_instance.main_ip]
}

resource "aws_route53_record" "AAAA" {
  zone_id = data.aws_route53_zone.cetacean_club.zone_id
  name    = "ingressd.${data.aws_route53_zone.cetacean_club.name}"
  type    = "AAAA"
  ttl     = "300"
  records = [resource.vultr_instance.my_instance.v6_main_ip]
}
