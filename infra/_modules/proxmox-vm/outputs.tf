# TODO wtf
# Build MAC-derived EUI-64 suffix per VM, no bitwise ops used
locals {
  eui64_suffix_by_node = {
    for node in proxmox_virtual_environment_vm.main :
    node.id => (
      # Parse MAC -> 6 bytes
      # Example: BC:24:11:EB:01:83
      # Flip the U/L bit (bit 1) of the first byte: if it’s 0 add 2, else subtract 2
      # Then insert ff:fe and format into 4x16-bit hex groups.
      # Result suffix for example above: be24:11ff:feeb:183
      (
        # lowercase, split, parse
        # keep these short to avoid paren hell
        # b = bytes[0..5], b0f = flipped first byte
        # words w1..w4 form the IID
        # join as hex groups
        # (all Terraform/HCL functions; no bitwise)
        # --
        # Precompute the bytes list
        # NOTE: locals inside expressions are not supported, so repeat split/parse where needed
        #
        # first byte (original)
        # b0 = parseint(split(":", lower(node.network_device[0].mac_address))[0], 16)
        # bit1 = floor(b0/2) % 2  -> 0 or 1
        # b0f = bit1 == 0 ? b0 + 2 : b0 - 2
        #
        join(":", [
          format(
            "%x",
            (
              (
                ((floor(parseint(split(":", lower(node.network_device[0].mac_address))[0], 16) / 2)) % 2) == 0
              )
              ? parseint(split(":", lower(node.network_device[0].mac_address))[0], 16) + 2
              : parseint(split(":", lower(node.network_device[0].mac_address))[0], 16) - 2
            ) * 256
            + parseint(split(":", lower(node.network_device[0].mac_address))[1], 16)
          ),
          format(
            "%x",
            parseint(split(":", lower(node.network_device[0].mac_address))[2], 16) * 256 + 255
          ),
          format(
            "%x",
            254 * 256 + parseint(split(":", lower(node.network_device[0].mac_address))[3], 16)
          ),
          format(
            "%x",
            parseint(split(":", lower(node.network_device[0].mac_address))[4], 16) * 256
            + parseint(split(":", lower(node.network_device[0].mac_address))[5], 16)
          )
        ])
      )
    )
  }
}

# Pick the IPv6 whose IID matches the MAC-derived EUI-64; fallback to first global (non-loopback, non-link-local)
output "hosts" {
  value = {
    for node in proxmox_virtual_environment_vm.main : node.name => {
      ipv6_address = (
        length([
          for ip in flatten(node.ipv6_addresses) :
          ip if endswith(lower(ip), local.eui64_suffix_by_node[node.id])
        ]) > 0
        ? [
          for ip in flatten(node.ipv6_addresses) :
          ip if endswith(lower(ip), local.eui64_suffix_by_node[node.id])
        ][0]
        : [
          for ip in flatten(node.ipv6_addresses) :
          ip if ip != "::1" && !startswith(lower(ip), "fe80:")
        ][0]
      )
    }
  }
}
