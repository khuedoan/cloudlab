output "ipv6_addresses" {
  value = { for node in hcloud_server.nodes : node.name => node.ipv6_address }
}
