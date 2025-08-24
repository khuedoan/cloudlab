terraform {
  required_version = "~> 1.8"

  required_providers {
    kubernetes = {
      source  = "hashicorp/kubernetes"
      version = "~> 2.37.1"
    }
  }
}

provider "kubernetes" {
  host                   = var.credentials.host
  client_certificate     = var.credentials.client_certificate
  client_key             = var.credentials.client_key
  cluster_ca_certificate = var.credentials.cluster_ca_certificate
}
