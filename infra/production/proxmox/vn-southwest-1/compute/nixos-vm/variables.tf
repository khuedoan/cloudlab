variable "name" {
  type = string
}

variable "node_name" {
  type    = string
  default = "proxmox"
}

variable "nixos" {
  type = object({
    flake = string
    host  = string
  })
}

variable "cpu" {
  type = object({
    cores = number
  })
}

variable "memory" {
  type = object({
    dedicated = number
  })
}

variable "cdrom" {
  type = object({
    file = string
  })

  default = {
    file = "nixos-24.11.20250123.035f8c0-x86_64-linux.iso"
  }
}

variable "disks" {
  type = map(object({
    size = number
  }))
}

variable "tags" {
  type    = list(string)
  default = []
}
