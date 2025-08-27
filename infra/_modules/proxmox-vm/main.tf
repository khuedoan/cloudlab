resource "proxmox_virtual_environment_vm" "main" {
  for_each = var.hosts

  name      = each.key
  node_name = var.node_name

  cpu {
    cores = each.value.cpu
  }

  memory {
    dedicated = 1024 * each.value.memory
    # Set floating to the same value as dedicated to enable ballooning device
    floating = 1024 * each.value.memory
  }

  cdrom {
    enabled   = true
    file_id   = "local:iso/${var.cdrom.file}"
    interface = "ide3"
  }

  disk {
    datastore_id = "local-lvm"
    interface    = "scsi0"
    size         = each.value.disk
    file_format  = "raw"
  }

  boot_order = [
    "scsi0",
    "ide3",
  ]

  bios = "ovmf"

  operating_system {
    type = "l26"
  }

  network_device {
    bridge = "vmbr0"
  }

  agent {
    enabled = true
  }

  tags = var.tags
}

# Temporary hack to wait for IP addresses to be actually available
# https://github.com/bpg/terraform-provider-proxmox/issues/776
resource "time_sleep" "wait_for_ip" {
  depends_on = [proxmox_virtual_environment_vm.main]

  create_duration = "30s"
}
