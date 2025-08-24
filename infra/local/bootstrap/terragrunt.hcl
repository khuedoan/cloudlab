include "root" {
  path = find_in_parent_folders("root.hcl")
}

terraform {
  source = "${find_in_parent_folders("_modules")}//bootstrap"
}

dependency "cluster" {
  config_path = "../cluster"
}

inputs = {
  cluster = dependency.cluster.outputs.name
  credentials = {
    host                   = dependency.cluster.outputs.credentials.host
    client_certificate     = dependency.cluster.outputs.credentials.client_certificate
    client_key             = dependency.cluster.outputs.credentials.client_key
    cluster_ca_certificate = dependency.cluster.outputs.credentials.cluster_ca_certificate
  }
  platform = "k3d"
}
