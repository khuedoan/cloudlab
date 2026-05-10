include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders("_modules")}//hetzner-metal"
}

inputs = {
  hosts = {
    "hetzner-metal-1" = {
      ipv6_address      = "2a01:4f9:3a:20ef::2"
      network_interface = "eth0"
      disk              = "/dev/sda"
    }
  }
}
