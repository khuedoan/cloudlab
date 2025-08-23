include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "./nixos-vm"
}

inputs = {
  name = "k3s"
  nixos = {
    flake = "${get_terragrunt_dir()}/../../../../nixos"
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
