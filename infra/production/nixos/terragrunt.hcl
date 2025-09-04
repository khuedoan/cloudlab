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

dependency "hetzner" {
  config_path = "../hetzner/compute"
}

inputs = {
  flake = "${find_in_parent_folders("_modules")}//nixos"
  hosts = merge(
    dependency.proxmox.outputs.hosts,
    dependency.hetzner.outputs.hosts,
  )
  kube_api_host = dependency.proxmox.outputs.hosts["production-master-1"].ipv6_address
  sops_file     = find_in_parent_folders("secrets.yaml")
}
