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

resource "vault_kubernetes_auth_backend_config" "k8s" {
  backend         = vault_auth_backend.kubernetes.path
  kubernetes_host = "https://kubernetes.default.svc.cluster.local"
}

#############
# App level #
#############

resource "vault_policy" "internal_app" {
  name   = "internal-app"
  policy = <<EOT
path "secret/data/db-pass" {
  capabilities = ["read"]
}
EOT
}

resource "vault_kubernetes_auth_backend_role" "database" {
  backend   = vault_auth_backend.kubernetes.path
  role_name = "database"
  bound_service_account_names = [
    "webapp-sa"
  ]
  bound_service_account_namespaces = [
    "default"
  ]
  token_ttl = 60 * 20
  token_policies = [
    vault_policy.internal_app.name
  ]
}

# TODO remove, just testing
resource "vault_generic_secret" "example" {
  path = "${vault_mount.secret.path}/db-pass"

  data_json = jsonencode({
    "password":   "db-secret-password",
  })
}
