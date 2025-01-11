data "aws_route53_zone" "within_website" {
  name = "within.website."
}

data "civo_ssh_key" "shiroko" {
  name = "shiroko"
}

data "civo_disk_image" "ubuntu" {
  filter {
    key    = "name"
    values = ["ubuntu-noble"]
  }
}

resource "civo_network" "pds" {
  label = "bsky-pds"
}

resource "civo_firewall" "pds" {
  name                 = "bsky-pds"
  network_id           = civo_network.pds.id
  create_default_rules = false

  ingress_rule {
    label      = "yolo"
    protocol   = "tcp"
    port_range = "1-65535"
    cidr       = ["0.0.0.0/0"]
    action     = "allow"
  }

  ingress_rule {
    label      = "yolo-udp"
    protocol   = "udp"
    port_range = "1-65535"
    cidr       = ["0.0.0.0/0"]
    action     = "allow"
  }

  egress_rule {
    label      = "yolo"
    protocol   = "tcp"
    port_range = "1-65535"
    cidr       = ["0.0.0.0/0"]
    action     = "allow"
  }

  egress_rule {
    label      = "yolo-udp"
    protocol   = "udp"
    port_range = "1-65535"
    cidr       = ["0.0.0.0/0"]
    action     = "allow"
  }
}

resource "civo_instance" "engram" {
  hostname    = "engram"
  tags        = ["xe", "pds"]
  notes       = "Bluesky PDS for pds.within.website"
  sshkey_id   = data.civo_ssh_key.shiroko.id
  firewall_id = civo_firewall.pds.id
  network_id  = civo_network.pds.id
  size        = "g4s.xsmall"
  disk_image  = data.civo_disk_image.ubuntu.diskimages[0].id
  script      = file("${path.module}/assimilate.sh")
  volume_type = "ms-xfs-2-replicas"
}

resource "aws_route53_record" "engram-within-website--A" {
  zone_id = data.aws_route53_zone.within_website.zone_id
  name    = "engram.${data.aws_route53_zone.within_website.name}"
  type    = "A"
  ttl     = "3600"
  records = [civo_instance.engram.public_ip]
}

resource "aws_route53_record" "star-engram-within-website--A" {
  zone_id = data.aws_route53_zone.within_website.zone_id
  name    = "*.engram.${data.aws_route53_zone.within_website.name}"
  type    = "A"
  ttl     = "3600"
  records = [civo_instance.engram.public_ip]
}
