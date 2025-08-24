terraform {
  required_version = "~> 1.5"

  required_providers {
    oci = {
      source  = "hashicorp/oci"
      version = "~> 6.0"
    }
  }
}
