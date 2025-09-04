include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders("_modules")}//proxmox-vm"
}

inputs = {
  hosts = {
    "kube-1" = { cpu = 4, memory = 12, disk = 128 }
  }

  tags = [
    "kube-staging"
  ]
}
