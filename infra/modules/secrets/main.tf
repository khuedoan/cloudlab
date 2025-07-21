locals {
  sources = {
    for k, v in var.sources : k => v.random ? random_string.random[k].result : v.value
  }
}

resource "random_string" "random" {
  for_each = { for k, v in var.sources : k => v if v.random }

  length = 128
}

resource "kubernetes_secret" "secret" {
  for_each = var.destinations

  metadata {
    namespace = split("/", each.key)[0]
    name      = split("/", each.key)[1]
    labels = {
      "app.kubernetes.io/managed-by" = "Terraform"
    }
  }

  type = each.value.type
  data = {
    for k, v in each.value.data : k => local.sources[v]
  }
}
