bind {
  http    = ":3034"
  https   = ":3035"
  metrics = ":3036"
}

autocert {
  s3_bucket          = "your-cert-bucket"
  s3_prefix          = "sakurajima/certs"
  email              = "your-email@example.com"
  http_redirect_code = 302 # Optional, defaults to 301

  # For Let's Encrypt staging, use this URL:
  directory_url = "https://acme-staging-v02.api.letsencrypt.org/directory"
}

logging {
  access_log = "./var/access.log"
  compress   = true

  filter "no-listening" {
    expression = <<EOF
      msg == "listening"
    EOF
  }

  # filter "no-http" {
  #   expression = <<EOF
  #     "for" in attrs && attrs["for"] == "http"
  #   EOF
  # }
}

domain "sakurajima.local.cetacean.club" {
  tls {
    cert = "./var/sakurajima.local.cetacean.club.pem"
    key  = "./var/sakurajima.local.cetacean.club-key.pem"
  }

  target        = "http://localhost:3000"
  health_target = "http://localhost:3036/healthz"

  # Optional: Request size limits to prevent DoS attacks
  # If not specified, defaults are: max_request_body=10MB, max_header_size=1MB, max_header_count=100
  limits {
    max_request_body = "10MB"    # Maximum request body size (e.g., "10MB", "1GB")
    max_header_size  = "1MB"     # Maximum size of headers (e.g., "1MB", "512KB")
    max_header_count = 100       # Maximum number of headers allowed
  }

  # HTTP timeout configuration to prevent hanging connections.
  # All values are human-readable durations (e.g., "5s", "100ms", "1m").
  # If omitted, sensible defaults are used:
  # - dial: 5s (time to establish connection)
  # - response_header: 10s (time to wait for response headers)
  # - idle: 90s (time to keep idle connections alive)
  timeouts {
    dial            = "5s"
    response_header = "10s"
    idle            = "90s"
  }
}
