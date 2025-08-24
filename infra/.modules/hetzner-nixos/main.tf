resource "hcloud_ssh_key" "main" {
  # TODO better key gen
  name       = "workstation"
  public_key = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIN5ue4np7cF34f6dwqH1262fPjkowHQ8irfjVC156PCG"
}

resource "hcloud_server" "nodes" {
  for_each = var.nodes

  name        = each.key
  server_type = "cax11"
  public_net {
    ipv4_enabled = true
    ipv6_enabled = true
  }
  image    = "debian-13" # Only used to bootstrap nixos-anywhere
  location = each.value.location
  ssh_keys = [
    hcloud_ssh_key.main.id
  ]
}

module "nixos" {
  for_each = hcloud_server.nodes

  source                 = "github.com/nix-community/nixos-anywhere//terraform/all-in-one"
  nixos_system_attr      = "${var.flake}#nixosConfigurations.master.config.system.build.toplevel"
  nixos_partitioner_attr = "${var.flake}#nixosConfigurations.master.config.system.build.diskoScript"
  target_host            = each.value.ipv6_address
  instance_id            = each.value.name
  build_on_remote        = true
  # extra_files_script     = "${path.module}/decrypt-ssh-secrets.sh"
  # disk_encryption_key_scripts = [{
  #   path   = "/tmp/secret.key"
  #   # script is below
  #   script = "${path.module}/decrypt-zfs-key.sh"
  # }]
}
