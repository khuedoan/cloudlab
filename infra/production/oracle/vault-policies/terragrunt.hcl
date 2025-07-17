include "root" {
  path = find_in_parent_folders("root.hcl")
}

terraform {
  source = "../../../modules//vault-policies"
}

# TODO wait for Vault API or unseal hook

inputs = {
}
