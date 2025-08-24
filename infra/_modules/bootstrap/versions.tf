terraform {
  required_version = "~> 1.8"

  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "~> 2.17"
    }
    kubectl = {
      source  = "alekc/kubectl"
      version = "~> 2.0"
    }
  }
}

provider "helm" {
  kubernetes {
    host                   = var.credentials.host
    client_certificate     = var.credentials.client_certificate
    client_key             = var.credentials.client_key
    cluster_ca_certificate = var.credentials.cluster_ca_certificate
  }
}

provider "kubectl" {
  host                   = var.credentials.host
  client_certificate     = var.credentials.client_certificate
  client_key             = var.credentials.client_key
  cluster_ca_certificate = var.credentials.cluster_ca_certificate
  load_config_file       = false
}
