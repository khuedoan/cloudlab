output "hosts" {
  value = {
    for node in proxmox_virtual_environment_vm.main : node.name => {
      ipv6_address = node.ipv6_addresses[1][0]
    }
  }
}
