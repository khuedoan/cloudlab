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
    ipv4_enabled = false
    ipv6_enabled = true
  }
  # TODO NixOS
  image    = "debian-13"
  location = each.value.location
  ssh_keys = [
    hcloud_ssh_key.main.id
  ]
}
