locals {
  common_labels = {
    "app.kubernetes.io/managed-by" = "Terraform"
  }
  sync_policy = {
    automated = {
      prune    = true
      selfHeal = true
    }
    syncOptions = [
      "CreateNamespace=true",
      "ApplyOutOfSyncOnly=true",
      "ServerSideApply=true"
    ]
  }
}
