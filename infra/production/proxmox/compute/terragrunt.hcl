include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders("_modules")}//proxmox-vm"
}

inputs = {
  hosts = {
    "kube-1" = {
      cpu    = 2
      memory = 4
      disk   = 128
    }
    "kube-2" = {
      cpu    = 2
      memory = 4
      disk   = 128
    }
    "kube-3" = {
      cpu    = 8
      memory = 16
      disk   = 128
    }
  }
}
