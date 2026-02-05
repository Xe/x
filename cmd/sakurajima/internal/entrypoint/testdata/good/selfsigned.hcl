bind {
  http    = ":65530"
  https   = ":65531"
  metrics = ":65532"
}

logging {
  access_log = "/var/log/access.log"
}

domain "osiris.local.cetacean.club" {
  tls {
    cert = "./testdata/selfsigned.crt"
    key  = "./testdata/selfsigned.key"
  }

  target        = "http://localhost:3000"
  health_target = "http://localhost:9091/healthz"

  allow_private_target = true
}
