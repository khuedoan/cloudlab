################
# System level #
################

resource "vault_auth_backend" "kubernetes" {
  type = "kubernetes"
}

resource "vault_auth_backend" "userpass" {
  type = "userpass"
}

resource "vault_mount" "secret" {
  path = "secret"
  type = "kv-v2"
}

resource "vault_kubernetes_auth_backend_config" "kubernetes" {
  backend         = vault_auth_backend.kubernetes.path
  kubernetes_host = "https://kubernetes.default.svc.cluster.local"
}

########
# User #
########

# TODO remove, just testing
# resource "vault_generic_endpoint" "khuedoan" {
#   path                 = "auth/${vault_auth_backend.userpass.path}/users/khuedoan"
#   ignore_absent_fields = true
#   data_json = jsonencode({
#     token_policies = [
#       "default",
#     ],
#     password = "testing"
#   })
# }

#############
# App level #
#############

# TODO remove, just testing
resource "vault_policy" "kubernetes_default" {
  name   = "kubernetes-default"
  policy = file("${path.module}/policies/kubernetes_default.hcl")
}
resource "vault_kubernetes_auth_backend_role" "kubernetes_default" {
  backend   = vault_auth_backend.kubernetes.path
  role_name = "kubernetes-default"
  bound_service_account_names = [
    "webapp-sa"
  ]
  bound_service_account_namespaces = [
    "default"
  ]
  token_ttl = 60 * 20
  token_policies = [
    vault_policy.kubernetes_default.name
  ]
}

resource "vault_kv_secret_v2" "db-pass" {
  mount = vault_mount.secret.path
  name  = "default/webapp-sa"

  data_json = jsonencode({
    password = "db-secret-password"
  })
}
