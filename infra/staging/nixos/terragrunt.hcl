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

  # TODO maybe use SSH agent or something?
  # Currently using separate SSH keys for rescue OS and NixOS serves 2 benefits:
  # - Leaking one doesn't affect the other
  # - We can't accidentally reinstall when NixOS already installed
  install_ssh_key_file    = "${get_env("HOME")}/.ssh/id_hetzner_metal"
  deployment_ssh_key_file = "${get_env("HOME")}/.ssh/id_ed25519"
}
