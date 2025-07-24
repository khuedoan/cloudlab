resource "kubectl_manifest" "loki" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "loki"
      namespace  = helm_release.argocd.namespace
      finalizers = ["resources-finalizer.argocd.argoproj.io"]
      labels     = local.common_labels
    }
    spec = {
      project = "default"
      destination = {
        name      = "in-cluster"
        namespace = "monitoring"
      }
      syncPolicy = local.sync_policy
      source = {
        repoURL        = "https://grafana.github.io/helm-charts"
        chart          = "loki-stack"
        targetRevision = "2.10.2"
        helm = {
          valuesObject = {
            loki = {
              isDefault = false
            }
            serviceMonitor = {
              enabled = true
            }
          }
        }
      }
    }
  })
}
