resource "local_file" "vault_password" {
  content         = var.vault_password
  filename        = "${path.root}/vault_password"
  file_permission = "0600"
}

resource "local_file" "ssh_private_key" {
  content         = var.ssh_private_key
  filename        = "${path.root}/private.pem"
  file_permission = "0600"
}

resource "local_file" "inventory" {
  filename        = "${path.root}/inventory.yml"
  file_permission = "0644"
  content = yamlencode({
    k3s = {
      hosts = {
        "${var.instance_public_ip}" = {
          ansible_user                 = "ubuntu"
          ansible_ssh_private_key_file = abspath(local_file.ssh_private_key.filename)
        }
      }
    }
  })
}

resource "null_resource" "cluster" {
  triggers = {
    inventory = local_file.inventory.content_md5
  }

  provisioner "local-exec" {
    working_dir = path.root
    command     = "ansible-playbook --inventory ${local_file.inventory.filename} --vault-password-file ${local_file.vault_password.filename} ${path.module}/main.yml"
  }
}

data "local_file" "kubeconfig" {
  filename = "${path.root}/kubeconfig.yaml"

  depends_on = [null_resource.cluster]
}
