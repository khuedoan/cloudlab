include "root" {
  path   = find_in_parent_folders("root.hcl")
  expose = true
}

terraform {
  source = "${find_in_parent_folders("_modules")}//hetzner-metal"
}

inputs = {
  hosts = {
    "hetzner-metal-1" = {
      ipv6_address      = "2a01:4f9:3a:20ef::2"
      network_interface = "enp35s0"
      # Use a stable disk path because /dev/sdX names can change across rescue, installer, and final boots.
      disk = "/dev/disk/by-id/ata-WDC_WD2000FYYZ-01UL1B1_WD-WCC1P0935477"
    }
  }
}
