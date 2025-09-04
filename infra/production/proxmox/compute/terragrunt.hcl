include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders("_modules")}//proxmox-vm"
}

inputs = {
  hosts = {
    # Masters are pets, workers are cattle, hence worker names are random
    "production-master-1" = { cpu = 4, memory = 12, disk = 128 }
    "production-aGVsbG8K" = { cpu = 4, memory = 12, disk = 128 }
    "production-d29ybGQK" = { cpu = 4, memory = 12, disk = 128 }
  }

  tags = [
    "kube-production"
  ]
}
