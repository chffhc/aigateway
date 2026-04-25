variable "IMAGE" {
  default = "aigateway"
}

variable "TAG" {
  default = "latest"
}

group "default" {
  targets = ["local"]
}

target "common" {
  context    = "."
  dockerfile = "Dockerfile"
  args = {
    GO_VERSION     = "1.22"
    ALPINE_VERSION = "3.19"
  }
}

target "local" {
  inherits  = ["common"]
  tags      = ["${IMAGE}:${TAG}"]
  platforms = ["linux/amd64"]
  load      = true
}

target "release" {
  inherits  = ["common"]
  tags      = ["${IMAGE}:${TAG}"]
  platforms = ["linux/amd64", "linux/arm64"]
}
