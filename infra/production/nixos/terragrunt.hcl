include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders("_modules")}//nixos"
}

dependency "proxmox" {
  config_path = "../proxmox/compute"
}

inputs = {
  flake = "${find_in_parent_folders("_modules")}//nixos"
  hosts = dependency.proxmox.outputs.hosts
}
