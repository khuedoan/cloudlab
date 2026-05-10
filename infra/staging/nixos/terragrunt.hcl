include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders("_modules")}//nixos"
}

dependency "hetzner_metal" {
  config_path = "../hetzner-metal"
}

inputs = {
  flake     = "${find_in_parent_folders("_modules")}//nixos"
  hosts     = dependency.hetzner_metal.outputs.hosts
  sops_file = find_in_parent_folders("secrets.yaml")
}
