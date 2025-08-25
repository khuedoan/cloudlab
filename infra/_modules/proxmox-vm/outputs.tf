output "ipv6_address" {
  value = proxmox_virtual_environment_vm.main.ipv6_addresses[1][0]
}
