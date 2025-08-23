include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = find_in_parent_folders(".modules")//cluster"
}

dependency "legacy" {
  config_path = "../legacy"
}

inputs = {
  name               = "production"
  instance_public_ip = dependency.legacy.outputs.instance_public_ip
  ssh_private_key    = dependency.legacy.outputs.ssh_private_key
  vault_password     = include.root.locals.secrets.vault_password
}
