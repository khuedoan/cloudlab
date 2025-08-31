output "credentials" {
  value = {
    client_certificate     = base64decode(yamldecode(data.external.kubeconfig.result.kubeconfig).users[0].user["client-certificate-data"])
    client_key             = base64decode(yamldecode(data.external.kubeconfig.result.kubeconfig).users[0].user["client-key-data"])
    cluster_ca_certificate = base64decode(yamldecode(data.external.kubeconfig.result.kubeconfig).clusters[0].cluster["certificate-authority-data"])
    host                   = yamldecode(data.external.kubeconfig.result.kubeconfig).clusters[0].cluster["server"]
  }

  sensitive = true
}
