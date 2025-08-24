variable "nodes" {
  type = map(object({
    location = string
  }))
}

variable "nixos" {
  type = object({
    flake = string
    host = string
  })
}
