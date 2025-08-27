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
    file = "nixos-minimal-25.05.808723.b1b329146965-x86_64-linux.iso"
  }
}

variable "tags" {
  type    = list(string)
  default = []
}
