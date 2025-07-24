variable "tenancy_id" {
  description = "The ID of the tenancy (same with the root compartment ID)"
  type        = string
}

variable "compartment_name" {
  description = "Name of the compartment where to create all resources"
  type        = string
}

variable "compartment_description" {
  description = "Description of the compartment where to create all resources"
  type        = string
}
