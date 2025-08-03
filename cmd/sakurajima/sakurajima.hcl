bind {
  http    = ":3034"
  https   = ":3035"
  metrics = ":3036"
}

logging {
  access_log = "./var/access.log"
  compress   = true
}

domain "sakurajima.local.cetacean.club" {
  tls {
    cert = "./var/sakurajima.local.cetacean.club.pem"
    key  = "./var/sakurajima.local.cetacean.club-key.pem"
  }

  target        = "http://localhost:3000"
  health_target = "http://localhost:3036/healthz"
}
