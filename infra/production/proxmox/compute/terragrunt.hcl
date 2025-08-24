include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders(".modules")}//proxmox-nixos"
}

inputs = {
  name = "k3s"
  nixos = {
    flake = "${find_in_parent_folders(".modules")}/nixos"
    host  = "k3s"
  }
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
