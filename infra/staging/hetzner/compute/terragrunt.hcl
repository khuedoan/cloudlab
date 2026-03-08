include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders("_modules")}//hetzner-vm"
}

inputs = {
  nodes = {
    "kube-4" = {
      location = "hel1"
    }
    # "kube-5" = {
    #   location = "nbg1"
    # }
  }
}
