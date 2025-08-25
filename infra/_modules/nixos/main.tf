module "nixos" {
  for_each = var.hosts

  source                 = "git::https://github.com/nix-community/nixos-anywhere//terraform/all-in-one?ref=main"
  nixos_system_attr      = "${var.flake}#nixosConfigurations.${each.key}.config.system.build.toplevel"
  nixos_partitioner_attr = "${var.flake}#nixosConfigurations.${each.key}.config.system.build.diskoScript"
  target_host            = each.value.ipv6_address
  instance_id            = each.key
}
