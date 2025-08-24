include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders(".modules")}//legacy"
}

inputs = {
  tenancy_ocid   = include.root.locals.secrets.oracle_tenancy_ocid
  vault_password = include.root.locals.secrets.vault_password
}
