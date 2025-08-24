variable "compartment_id" {
  description = "Compartment ID where to create all resources"
  type        = string
}

variable "vcn_cidr_blocks" {
  description = "The list of IPv4 CIDR blocks the VCN will use"
  type        = list(string)
  default = [
    "10.0.0.0/16"
  ]
}

variable "subnet_cidr_block" {
  type    = string
  default = "10.0.0.0/24"
}
