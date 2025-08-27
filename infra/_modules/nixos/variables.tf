variable "flake" {
  type = string
}

variable "hosts" {
  type = map(object({
    ipv6_address = string
  }))
}

variable "sops_file" {
  type = string
}
