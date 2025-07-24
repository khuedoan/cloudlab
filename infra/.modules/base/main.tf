data "oci_identity_tenancy" "tenancy" {
  tenancy_id = var.tenancy_id
}

resource "oci_identity_compartment" "compartment" {
  compartment_id = data.oci_identity_tenancy.tenancy.id
  name           = var.compartment_name
  description    = var.compartment_description
}
