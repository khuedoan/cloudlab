resource "kubectl_manifest" "prometheus" {
  server_side_apply = true
  yaml_body = yamlencode({
    apiVersion = "argoproj.io/v1alpha1"
    kind       = "Application"
    metadata = {
      name       = "prometheus"
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
        repoURL        = "https://prometheus-community.github.io/helm-charts"
        chart          = "kube-prometheus-stack"
        targetRevision = "72.0.1"
        helm = {
          valuesObject = {
            valuesObject = {
              grafana = {
                additionalDataSources = [
                  {
                    name = "Loki"
                    type = "loki"
                    url  = "http://loki:3100"
                  },
                ]
                enabled                = false
                forceDeployDashboards  = true
                forceDeployDatasources = true
              }
              prometheus = {
                prometheusSpec = {
                  podMonitorSelectorNilUsesHelmValues     = false
                  probeSelectorNilUsesHelmValues          = false
                  ruleSelectorNilUsesHelmValues           = false
                  serviceMonitorSelectorNilUsesHelmValues = false
                }
              }
            }
          }
        }
      }
    }
  })
}
