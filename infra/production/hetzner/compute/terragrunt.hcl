include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders("_modules")}//hetzner-vm"
}

inputs = {
  nodes = {
    # Masters are pets, workers are cattle, hence worker names are random
    "production-YnJ1aGgK" = {
      location = "hel1"
    }
  }
}
