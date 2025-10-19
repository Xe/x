variable "VERSION" { default = "devel" }

group "default" {
  targets = [
    "aura",
    "falin",
    "mimi",
    "mi",
    "python-wasm-mcp",
    "httpdebug",
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

target "mimi" {
  context = "."
  dockerfile = "./docker/mimi.Dockerfile"
  platforms = [
    "linux/amd64"
  ]
  pull = true
  tags = [
    "ghcr.io/xe/x/mimi:${VERSION}",
    "ghcr.io/xe/x/mimi:latest",
  ]
}

target "mi" {
  context = "."
  dockerfile = "./docker/mi.Dockerfile"
  platforms = [
    "linux/amd64",
  ]
  pull = true
  tags = [
    "ghcr.io/xe/x/mi:${VERSION}",
    "ghcr.io/xe/x/mi:latest",
  ]
}

target "httpdebug" {
  context = "."
  dockerfile = "./docker/httpdebug.Dockerfile"
  platforms = [
    "linux/amd64",
    "linux/arm64",
  ]
  pull = true
  tags = [
    "ghcr.io/xe/x/httpdebug:${VERSION}",
    "ghcr.io/xe/x/httpdebug:latest",
  ]
}

target "python-wasm-mcp" {
  context = "."
  dockerfile = "./docker/python-wasm-mcp.Dockerfile"
  platforms = [
    "linux/amd64",
  ]
  pull = true
  tags = [
    "ghcr.io/xe/x/python-wasm-mcp:${VERSION}",
    "ghcr.io/xe/x/python-wasm-mcp:latest",
  ]
}

target "sakurajima" {
  context = "."
  dockerfile = "./docker/sakurajima.Dockerfile"
  platforms = [
    "linux/amd64",
    "linux/arm64",
    "linux/ppc64le",
    "linux/riscv64"
  ]
  pull = true
  tags = [
    "ghcr.io/xe/x/sakurajima:${VERSION}",
    "ghcr.io/xe/x/sakurajima:latest",
  ]
}
