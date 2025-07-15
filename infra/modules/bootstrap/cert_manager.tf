resource "kubectl_manifest" "cert_manager" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "cert-manager"
      namespace  = helm_release.argocd.namespace
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      labels     = local.common_labels
    }
    spec = {
      project = "default"
      destination = {
        name      = "in-cluster"
        namespace = "cert-manager"
      }
      syncPolicy = local.sync_policy
      source = {
        repoURL        = "https://charts.jetstack.io"
        chart          = "cert-manager"
        targetRevision = "1.18.2"
        helm = {
          valuesObject = {
            crds = {
              enabled = true
            }
          }
        }
      }
    }
  })
}

resource "kubectl_manifest" "trust_manager" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "trust-manager"
      namespace  = helm_release.argocd.namespace
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      labels     = local.common_labels
    }
    spec = {
      project = "default"
      destination = {
        name      = "in-cluster"
        namespace = "cert-manager"
      }
      syncPolicy = local.sync_policy
      source = {
        repoURL        = "https://charts.jetstack.io"
        chart          = "trust-manager"
        targetRevision = "0.18.0"
      }
    }
  })
}
