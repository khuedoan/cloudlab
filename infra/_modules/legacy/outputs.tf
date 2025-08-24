output "instance_public_ip" {
  value = module.instance.public_ip
}

output "ssh_private_key" {
  value     = tls_private_key.ssh.private_key_openssh
  sensitive = true
}
