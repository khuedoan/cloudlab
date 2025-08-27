# TODO temporarily disable Hetzner until I fix the IPv6 issue
# https://wiki.nixos.org/wiki/Install_NixOS_on_Hetzner_Cloud
skip = true

include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders("_modules")}//hetzner-vm"
}

inputs = {
  nodes = {
    "master-1" = {
      location = "hel1"
    }
    # "worker-1" = {
    #   location = "nbg1"
    # }
    # "worker-2" = {
    #   location = "fsn1"
    # }
  }
}
