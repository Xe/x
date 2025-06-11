variable "VERSION" { default = "devel" }

group "default" {
  targets = [
    "aura",
    "falin",
  ]
}

target "aura" {
  context = "."
  dockerfile = "./docker/aura.Dockerfile"
  platforms = [
    "linux/amd64"
  ]
  pull = true
  tags = [
    "ghcr.io/xe/x/aura:${VERSION}",
    "ghcr.io/xe/x/aura:latest",
  ]
}

target "falin" {
  context = "./migroserbices/falin"
  dockerfile = "./Dockerfile"
  platforms = [
    "linux/amd64"
  ]
  pull = true
  tags = [
    "ghcr.io/xe/x/falin:${VERSION}",
    "ghcr.io/xe/x/falin:latest",
  ]
}