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
    file = "nixos-installer.iso"
  }
}

variable "tags" {
  type    = list(string)
  default = []
}
