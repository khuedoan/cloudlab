variable "hosts" {
  type = map(object({
    cpu    = number
    memory = number
    disk   = number
  }))
}

variable "node_name" {
  type    = string
  default = "proxmox"
}

variable "cdrom" {
  type = object({
    file = string
  })

  default = {
    file = "nixos-24.11.20250123.035f8c0-x86_64-linux.iso"
  }
}

variable "tags" {
  type    = list(string)
  default = []
}
