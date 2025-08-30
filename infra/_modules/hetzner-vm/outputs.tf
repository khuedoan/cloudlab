output "hosts" {
  value = {
    for node in hcloud_server.nodes : node.name => {
      ipv6_address = node.ipv6_address
    }
  }
}
