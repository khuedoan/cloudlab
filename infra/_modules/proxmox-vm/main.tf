resource "proxmox_virtual_environment_vm" "main" {
  name      = var.name
  node_name = var.node_name

  cpu {
    cores = var.cpu.cores
  }

  memory {
    dedicated = 1024 * var.memory.dedicated
    # Set floating to the same value as dedicated to enable ballooning device
    floating = 1024 * var.memory.dedicated
  }

  cdrom {
    enabled   = true
    file_id   = "local:iso/${var.cdrom.file}"
    interface = "ide3"
  }

  dynamic "disk" {
    for_each = var.disks
    content {
      datastore_id = "local-lvm"
      interface    = "scsi0"
      size         = disk.value.size
      file_format  = "raw"
    }
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
