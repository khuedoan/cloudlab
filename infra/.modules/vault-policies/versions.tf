terraform {
  required_providers {
    vault = {
      source  = "hashicorp/vault"
      version = "~> 5.1.0"
    }
  }
}

provider "vault" {
  # Configure this provider through the environment variables:
  # - VAULT_ADDR
  # - VAULT_TOKEN
}
