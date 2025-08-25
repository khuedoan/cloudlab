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
