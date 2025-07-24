data "oci_identity_availability_domains" "availability_domains" {
  compartment_id = var.compartment_id
}

data "oci_core_images" "image" {
  compartment_id           = var.compartment_id
  operating_system         = var.image.operating_system
  operating_system_version = var.image.version
  shape                    = var.shape.name
  sort_by                  = "TIMECREATED"
  sort_order               = "DESC"
}

resource "oci_core_instance" "instance" {
  compartment_id      = var.compartment_id
  display_name        = var.display_name
  availability_domain = data.oci_identity_availability_domains.availability_domains.availability_domains[0].name

  create_vnic_details {
    assign_public_ip = false
    subnet_id        = var.subnet_id
  }

  extended_metadata = {
    subnet_id = var.subnet_id
  }

  metadata = {
    ssh_authorized_keys = var.ssh_public_key
  }

  shape = var.shape.name

  source_details {
    boot_volume_size_in_gbs = var.boot_volume_size
    source_type             = "image"
    source_id               = data.oci_core_images.image.images[0].id
  }

  dynamic "shape_config" {
    for_each = length(var.shape.config) > 0 ? [1] : []

    content {
      ocpus         = tonumber(lookup(var.shape.config, "cpus", 0))
      memory_in_gbs = tonumber(lookup(var.shape.config, "memory", 0))
    }
  }

  lifecycle {
    ignore_changes = [
      metadata,
      source_details
    ]
  }
}

resource "oci_core_volume" "data" {
  compartment_id      = var.compartment_id
  display_name        = "${var.display_name}-data"
  availability_domain = data.oci_identity_availability_domains.availability_domains.availability_domains[0].name
  size_in_gbs         = var.data_volume_size
}

resource "oci_core_volume_attachment" "data" {
  instance_id     = oci_core_instance.instance.id
  volume_id       = oci_core_volume.data.id
  attachment_type = "paravirtualized"
}

data "oci_core_vnic_attachments" "vnic_attachments" {
  compartment_id = var.compartment_id
  instance_id    = oci_core_instance.instance.id
}

data "oci_core_vnic" "vnic" {
  vnic_id = lookup(data.oci_core_vnic_attachments.vnic_attachments.vnic_attachments[0], "vnic_id")
}

data "oci_core_private_ips" "private_ip" {
  vnic_id = data.oci_core_vnic.vnic.id
}

resource "oci_core_public_ip" "public_ip" {
  compartment_id = var.compartment_id
  lifetime       = "RESERVED"
  private_ip_id  = lookup(data.oci_core_private_ips.private_ip.private_ips[0], "id")
}
