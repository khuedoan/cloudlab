include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

dependency "cluster" {
  config_path = "../cluster"

  mock_outputs = {}
}

terraform {
  source = "../../../../modules//empty"
}

inputs = {}
