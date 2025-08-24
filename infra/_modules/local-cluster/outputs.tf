output "name" {
  value = var.name
}

output "credentials" {
  value     = k3d_cluster.this.credentials[0]
  sensitive = true
}
