variable "tenancy_ocid" {
  description = "The ID of the tenancy (same with the root compartment ID)"
  type        = string
}

variable "compartment_name" {
  description = "Name of the compartment where to create all resources"
  type        = string
  default     = "cloudlab"
}

variable "compartment_description" {
  description = "Description of the compartment where to create all resources"
  type        = string
  default     = "Cloudlab Project"
}
