terraform {
  # TODO move to modules
  source = "./hetzner-vm"
}

# TODO temp skip
skip = true

inputs = {
  nodes = {
    "master-1" = {
      location = "hel1"
    }
    # "worker-1" = {
    #   location = "nbg1"
    # }
    # "worker-2" = {
    #   location = "fsn1"
    # }
  }
  flake = get_terragrunt_dir()
}
