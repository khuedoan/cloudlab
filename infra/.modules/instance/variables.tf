variable "display_name" {
  description = "Display name of the instance"
  type        = string
}

variable "compartment_id" {
  description = "Compartment ID where to create all resources"
  type        = string
}

variable "shape" {
  description = "The shape configuration requested for the instance"
  type = object({
    name   = string
    config = map(any)
  })
  validation {
    condition     = !(can(regex("Flex", var.shape.name)) && length(var.shape.config) == 0)
    error_message = "Shape config not found. Shape Config is required while using flexible shapes."
  }
}

variable "subnet_id" {
  type = string
}

variable "ssh_public_key" {
  description = "SSH public key to add to the instance"
  type        = string
}

variable "boot_volume_size" {
  description = "The size of the boot volume in GBs"
  type        = number
  default     = 50

  validation {
    condition = (
      var.boot_volume_size >= 50 &&
      var.boot_volume_size <= 32768
    )
    error_message = "Minimum value is 50 GB and maximum value is 32,768 GB (32 TB)."
  }
}

variable "data_volume_size" {
  description = "The size of the data volume in GBs"
  type        = number
  default     = 150
}

variable "image" {
  description = "OS image properties"

  type = object({
    operating_system = string
    version          = string
  })

  default = {
    operating_system = "Canonical Ubuntu"
    version          = "20.04"
  }
}
