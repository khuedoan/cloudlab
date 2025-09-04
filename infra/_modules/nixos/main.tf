resource "local_file" "hosts" {
  content         = jsonencode(var.hosts) # Converts variables to JSON
  filename        = "${var.flake}/hosts.json"
  file_permission = "600"

  provisioner "local-exec" {
    interpreter = ["bash", "-c"]
    command     = "git add -f '${var.flake}/hosts.json'"
  }
}

module "nixos" {
  for_each = var.hosts

  source                 = "git::https://github.com/nix-community/nixos-anywhere//terraform/all-in-one?ref=main"
  nixos_system_attr      = "${var.flake}#nixosConfigurations.${each.key}.config.system.build.toplevel"
  nixos_partitioner_attr = "${var.flake}#nixosConfigurations.${each.key}.config.system.build.diskoScript"
  target_host            = each.value.ipv6_address
  instance_id            = each.key
  build_on_remote        = true
  extra_files_script     = "${path.module}/decrypt-age-keys.sh"
  extra_environment = {
    SOPS_FILE = var.sops_file
  }

  depends_on = [
    local_file.hosts
  ]
}

data "external" "kubeconfig" {
  program = ["${path.module}/kubeconfig_datasource.py"]

  query = {
    user = "root"
    host = var.kube_api_host
  }

  depends_on = [
    module.nixos
  ]
}
