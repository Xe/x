autocert {
  accept_tos = true
  email      = "me@xeiaso.net"

  s3 {
    bucket = "techaro-prod-certs"
    prefix = "sakurajima"
  }
}

bind {
  http    = ":3004"
  https   = ":3005"
  metrics = ":9091"
}

domain "sakurajima.local.cetacean.club" {
  tls {
    cert = "./var/sakurajima.local.cetacean.club.pem"
    key  = "./var/sakurajima.local.cetacean.club.key.pem"
  }

  route "/" {
    reverse_proxy {
      target        = "http://localhost:3000"
      health_target = "http://localhost:3002/healthz"
    }
  }

  route "/www/" {
    folder = "./var/www"
  }
}