variable "credentials" {
  type = object({
    client_certificate     = string
    client_key             = string
    cluster_ca_certificate = string
    host                   = string
  })
}

variable "sources" {
  type = map(object({
    random = optional(bool, false)
    value  = optional(string)
  }))
  default = {}
}

variable "destinations" {
  type = map(object({
    data = map(string)
    type = optional(string, "Opaque")
  }))
  default = {}
}
