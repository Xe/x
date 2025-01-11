data "aws_route53_zone" "cetacean_club" {
  name = "cetacean.club."
}

resource "tigris_bucket" "the-cetacean" {
  bucket = "the.cetacean.club"
}

resource "tigris_bucket_public_access" "the-cetacean" {
  bucket              = tigris_bucket.the-cetacean.bucket
  acl                 = "public-read"
  public_list_objects = false
}

resource "tigris_bucket_website_config" "the-cetacean" {
  bucket      = tigris_bucket.the-cetacean.bucket
  domain_name = tigris_bucket.the-cetacean.bucket
}

resource "aws_route53_record" "the-cetacean-club--CNAME" {
  zone_id = data.aws_route53_zone.cetacean_club.zone_id
  name    = tigris_bucket.the-cetacean.bucket
  type    = "CNAME"
  ttl     = "3600"
  records = ["${tigris_bucket.the-cetacean.bucket}.fly.storage.tigris.dev"]
}

resource "aws_route53_record" "_atproto_the_cetacean_club" {
  zone_id = data.aws_route53_zone.cetacean_club.zone_id
  name    = "_atproto.${tigris_bucket.the-cetacean.bucket}"
  type    = "TXT"
  ttl     = "3600"
  records = ["did=did:web:the.cetacean.club"]
}
