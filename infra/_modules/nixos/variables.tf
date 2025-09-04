variable "flake" {
  type = string
}

variable "hosts" {
  type = map(object({
    ipv6_address = string
  }))
}

variable "kube_api_host" {
  type = string
}

variable "sops_file" {
  type = string
}
