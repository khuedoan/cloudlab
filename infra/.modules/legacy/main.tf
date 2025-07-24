module "base" {
  source                  = "../base"
  tenancy_id              = var.tenancy_ocid
  compartment_name        = var.compartment_name
  compartment_description = var.compartment_description
}

module "network" {
  source         = "../network"
  compartment_id = module.base.compartment_id
}

resource "tls_private_key" "ssh" {
  algorithm = "ED25519"
}

module "instance" {
  source         = "../instance"
  compartment_id = module.base.compartment_id
  display_name   = "cloudlab"
  subnet_id      = module.network.subnet_id
  ssh_public_key = tls_private_key.ssh.public_key_openssh
  shape = {
    name = "VM.Standard.A1.Flex"
    config = {
      cpus   = 4
      memory = 24
    }
  }
}
