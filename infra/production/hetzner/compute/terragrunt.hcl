terraform {
  source = "${find_in_parent_folders("_modules")}//hetzner-nixos"
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
  nixos = {
    flake = "${find_in_parent_folders("_modules")}/nixos"
    host = "k3s"
  }
}
