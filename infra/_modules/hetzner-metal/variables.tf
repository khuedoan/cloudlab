variable "hosts" {
  type = map(object({
    ipv6_address      = string
    network_interface = string
    disk              = string
  }))
}
