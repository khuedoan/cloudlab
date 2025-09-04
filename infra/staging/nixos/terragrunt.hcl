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
  hosts = merge(
    dependency.proxmox.outputs.hosts,
  )
  sops_file = find_in_parent_folders("secrets.yaml")
}
