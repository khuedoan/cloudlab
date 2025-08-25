include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders("_modules")}//proxmox-vm"
}

inputs = {
  name = "k3s"
  cpu = {
    cores = 8
  }
  memory = {
    dedicated = 16
  }
  disks = {
    os = {
      size = 256
    }
  }
}
