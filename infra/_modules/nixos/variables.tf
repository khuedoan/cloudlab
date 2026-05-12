variable "flake" {
  type = string
}

variable "hosts" {
  type = map(object({
    ipv6_address      = string
    network_interface = string
    disk              = string
  }))
}

variable "sops_file" {
  type = string
}

variable "install_ssh_key_file" {
  type = string
}

variable "deployment_ssh_key_file" {
  type = string
}
