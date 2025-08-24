variable "name" {
  type = string
}

variable "instance_public_ip" {
  type = string
}

variable "ssh_private_key" {
  type      = string
  sensitive = true
}

variable "vault_password" {
  type      = string
  sensitive = true
}
